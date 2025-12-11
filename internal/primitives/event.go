// Event provides the immutable event primitive for statechart transitions.
//
// Events are value types designed for zero-allocation creation via stack allocation.
// Once created, Events should not be mutated. Use NewEvent for construction.
//
// # Immutability
//
// Event fields are exported for convenience in read-only contexts, but consumers MUST
// NOT modify them after construction. Violations break statechart guarantees.
//
// # Zero Allocation
//
// NewEvent returns a stack-allocated Event. The Data field holds any value (interface{}),
// but passing stack values (structs, primitives) avoids heap allocation.
//
// Example:
//
//	event := NewEvent("transition", MyPayload{Value: 42})
//	// Zero heap allocation if MyPayload is small (&lt;16 bytes typically)
package primitives

type Event struct {
	Type string
	Data any
}

// NewEvent creates and returns a new immutable Event.
//
// This is zero-heap-allocation when Data is a stack value (small structs, primitives).
// Returns Event by value for stack allocation and copy elision.
func NewEvent(eventType string, data any) Event {
	return Event{
		Type: eventType,
		Data: data,
	}
}
