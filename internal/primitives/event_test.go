package primitives

import "testing"

func TestNewEvent(t *testing.T) {
	e := NewEvent("test", 42)
	if e.Type != "test" {
		t.Errorf("got Type=%q want test", e.Type)
	}
	if v, ok := e.Data.(int); !ok || v != 42 {
		t.Errorf("got Data=%v (%T) want 42", e.Data, e.Data)
	}
}

func TestEventImmutability(t *testing.T) {
	e := NewEvent("test", 42)
	eCopy := e
	eCopy.Type = "modified"
	eCopy.Data = "changed"
	if e.Type != "test" {
		t.Error("original Type was mutated")
	}
	if v, ok := e.Data.(int); !ok || v != 42 {
		t.Error("original Data was mutated")
	}
}
