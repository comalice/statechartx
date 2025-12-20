package main

import (
	"context"
	"fmt"
	"os"

	. "github.com/comalice/statechartx"
)

func logBuilder(msg string) Action {
	return func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		fmt.Printf("%s", msg+"\n")
		return nil
	}
}

// ---

func main() {
	evtRun := Event{ID: 1}
	evtStop := Event{ID: 2}

	init := State{ID: 1}
	running := State{ID: 2}
	stopped := State{ID: 3}

	init.On(evtRun, &running, nil, nil)
	init.OnEntry(logBuilder("enter init"))
	init.OnExit(logBuilder("exit init"))
	running.On(evtStop, &stopped, nil, nil)
	running.OnEntry(logBuilder("enter running"))
	running.OnExit(logBuilder("exiting running"))
	stopped.On(evtRun, &running, nil, nil)
	stopped.OnEntry(logBuilder("enter stopped"))
	stopped.OnExit(logBuilder("exiting stopped"))

	machine, err := NewMachine(&init, &running, &stopped)
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	machine.Start(ctx)

	machine.Send(ctx, evtRun)
	machine.Send(ctx, evtStop)
	machine.Send(ctx, evtRun)

}
