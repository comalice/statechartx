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
	mb := primitives.NewMachineBuilder("traffic-light", "traffic")
	traffic := mb.Compound("traffic").WithInitial("red")
	traffic.Atomic("red").Transition("TIMER", "green")
	traffic.Atomic("green").Transition("TIMER", "yellow")
	traffic.Atomic("yellow").Transition("TIMER", "red")

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
	for {
		select {
		case <-ticker.C:
			event := primitives.NewEvent("TIMER", nil)
			if err := m.Send(event); err != nil {
				fmt.Printf("Send error: %v\n", err)
			}
			fmt.Printf("\n--- Cycle %d ---\n", cycles+1)
			fmt.Println("Current states:", m.Current())
			fmt.Println("DOT:\n" + m.Visualize())
			// Demo publish consumption
			select {
			case pubEvent := <-publishChan:
				fmt.Printf("Published: %s (%s)\n", pubEvent.Metadata.Transition, pubEvent.Event.Type)
			default:
			}
			cycles++
			if cycles >= 12 {
				fmt.Println("Demo complete after 12 cycles.")
				return
			}
		case <-sig:
			fmt.Println("\nShutting down gracefully...")
			return
		}
	}
}
