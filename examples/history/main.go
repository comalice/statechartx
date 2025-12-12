package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
	"github.com/comalice/statechartx/internal/production"
)

func main() {
	// History: session > sub (compound) > a/b with history h
	mb := primitives.NewMachineBuilder("history", "session")
	session := mb.Compound("session").WithInitial("sub")
	sub := session.Compound("sub").WithInitial("a")
	sub.Atomic("a").Transition("SWITCH", "b")
	sub.History("h", true)
	b := sub.Atomic("b")
	b.Transition("SAVE", "h")
	b.Transition("RESTORE", "h")
	session.Atomic("restore").Transition("LOAD", "sub")

	config := mb.Build()

	persister, err := production.NewJSONPersister("/tmp")
	if err != nil {
		panic(err)
	}
	publishChan := make(chan production.PublishedEvent, 100)
	publisher := production.NewChannelPublisher(publishChan)
	visualizer := &production.DefaultVisualizer{}

	m := core.NewMachine(config,
		core.WithPersister(persister),
		core.WithPublisher(publisher),
		core.WithVisualizer(visualizer),
	)

	if err := m.Start(); err != nil {
		panic(err)
	}
	defer m.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	cycles := 0
	events := []string{"SWITCH", "SAVE", "RESTORE", "SWITCH", "LOAD"}
	for cycles < 5 {
		select {
		case <-ticker.C:
			ev := primitives.NewEvent(events[cycles%len(events)], nil)
			if err := m.Send(ev); err != nil {
				fmt.Printf("Send error: %v\n", err)
			}
			fmt.Printf("\n--- Cycle %d (%s) ---\n", cycles+1, ev.Type)
			fmt.Println("Current:", m.Current())
			fmt.Println("DOT:\n" + m.Visualize())
			select {
			case pub := <-publishChan:
				fmt.Printf("Published: %s (%s)\n", pub.Metadata.Transition, pub.Event.Type)
			default:
			}
			cycles++
		case <-sig:
			return
		}
	}
}
