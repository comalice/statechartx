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
	// Parallel regions: ui > left/right independent
	mb := primitives.NewMachineBuilder("parallel", "ui")
	ui := mb.Compound("ui").WithInitial("regions")
	regions := ui.Parallel("regions").WithInitial("left")
	regions.Atomic("left").Transition("LCLICK", "right")
	regions.Atomic("right").Transition("RCLICK", "left")

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
	for cycles < 8 {
		select {
		case <-ticker.C:
			evType := "LCLICK"
			if cycles%2 == 1 {
				evType = "RCLICK"
			}
			if err := m.Send(primitives.NewEvent(evType, nil)); err != nil {
				fmt.Printf("Send error: %v\n", err)
			}
			fmt.Printf("\n--- Cycle %d (%s) ---\n", cycles+1, evType)
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
