# Statechartx Builder

A fluent, type-safe builder for constructing complex statecharts in Go. This package implements the **Functional Options Pattern** to provide a declarative API for state machine definition.

## Features

* **Declarative Syntax**: Define states, transitions, and guards in a single, readable block.
* **Hierarchical Support**: Easily create composite (nested) states.
* **Type-Safe Guards & Actions**: Leverage Go's static typing for state machine logic.
* **Middleware Ready**: Extensible through functional options (e.g., adding logging or telemetry).

## Installation

```bash
go get github.com/youruser/yourproject/builder

```

## Quick Start

```go
package main

import (
    "context"
    "github.com/youruser/yourproject/builder"
)

func main() {
    // Define a simple leaf state with entry/exit actions
    idle := builder.New("IDLE",
        builder.WithEntry(func(ctx context.Context, e event, from, to ID, ext any) {
            println("Entering IDLE")
        }),
        builder.On("START", "RUNNING"),
    )

    // Define a composite state
    // The first child ("IDLE") is automatically set as the initial state
    root := builder.Composite("ROOT",
        idle,
        builder.New("RUNNING",
            builder.On("STOP", "IDLE"),
            builder.On("ERROR", "FAULT", builder.WithGuard(isRecoverable)),
        ),
    )
}

```

## Patterns Used

### Functional Options

Instead of telescoping constructors, we use functions that return `Option` closures. This allows for clean defaults and easy expansion:

```go
// Python equivalent: New("IDLE", entry=my_func)
builder.New("IDLE", builder.WithEntry(myFunc))

```

### Transition Sub-Builders

The `On` function accepts `TransOption` arguments, allowing you to attach guards and actions specifically to a transition without polluting the state configuration.

## API Reference

| Function                            | Description                                              |
|-------------------------------------|----------------------------------------------------------|
| `New(id, ...Option)`                | Creates a leaf state.                                    |
| `Composite(id, ...*State)`          | Creates a state containing children; index 0 is initial. |
| `WithEntry(Action)`                 | Option to set the entry hook.                            |
| `WithExit(Action)`                  | Option to set the exit hook.                             |
| `On(event, target, ...TransOption)` | Adds a transition to a state.                            |
| `WithGuard(Guard)`                  | Transition option to add conditional logic.              |
| `WithAction(Action)`                | Transition option to add action.                         |
