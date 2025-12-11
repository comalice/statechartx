package core

import (
	"sync"
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/primitives"
)

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMachine_StartInitialState(t *testing.T) {
	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic),
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	m := NewMachine(config)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	want := []string{"idle"}
	if got := m.Current(); !equalStringSlices(got, want) {
		t.Errorf("Current() = %v, want %v", got, want)
	}
}

func TestMachine_BasicTransitions(t *testing.T) {
	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"start": {{Target: "active"}},
				}),
			"active": primitives.NewStateConfig("active", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"stop": {{Target: "idle"}},
				}),
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	m := NewMachine(config)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	// idle -> active
	if err := m.Send(primitives.NewEvent("start", nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // allow processing
	if got, want := m.Current(), []string{"active"}; !equalStringSlices(got, want) {
		t.Errorf("after 'start' Current() = %v, want %v", got, want)
	}

	// active -> idle
	if err := m.Send(primitives.NewEvent("stop", nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if got, want := m.Current(), []string{"idle"}; !equalStringSlices(got, want) {
		t.Errorf("after 'stop' Current() = %v, want %v", got, want)
	}
}

func TestMachine_HierarchicalTransitions(t *testing.T) {
	parent := primitives.NewStateConfig("parent", primitives.Compound).
		WithInitial("child1").
		WithChildren([]*primitives.StateConfig{
			primitives.NewStateConfig("child1", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"switch": {{Target: "parent.child2"}},
				}),
			primitives.NewStateConfig("child2", primitives.Atomic),
		})

	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "parent",
		States: map[string]*primitives.StateConfig{
			"parent": parent,
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	m := NewMachine(config)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	// Initial resolves to parent.child1
	time.Sleep(50 * time.Millisecond)
	if got, want := m.Current(), []string{"parent.child1"}; got[0] != want[0] {
		t.Errorf("initial Current() = %v, want %v", got, want)
	}

	// child1 -> child2
	if err := m.Send(primitives.NewEvent("switch", nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if got, want := m.Current(), []string{"parent.child2"}; got[0] != want[0] {
		t.Errorf("after switch = %v, want %v", got, want)
	}
}

func TestMachine_EventQueueBackpressure(t *testing.T) {
	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic),
		},
	}
	m := NewMachine(config, WithQueueSize(5)) // small queue
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	// Fill queue
	for i := 0; i < 5; i++ {
		if err := m.Send(primitives.NewEvent("tick", nil)); err != nil {
			t.Errorf("Send %d failed: %v", i, err)
		}
	}

	// Backpressure
	if err := m.Send(primitives.NewEvent("overflow", nil)); err == nil {
		t.Error("expected backpressure error, got nil")
	} else if err.Error() != "event queue full (backpressure)" {
		t.Errorf("wrong error: %v", err)
	}
}

func TestMachine_GracefulShutdown(t *testing.T) {
	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic),
		},
	}
	m := NewMachine(config)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}

	m.Stop()

	// Send after stop should enqueue but not process
	if err := m.Send(primitives.NewEvent("poststop", nil)); err != nil {
		t.Errorf("Send after stop: %v", err)
	}

	// Multiple Stop idempotent
	m.Stop()
}

func TestMachine_ConcurrentSend(t *testing.T) {
	config := primitives.MachineConfig{
		ID:      "test",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"go": {{Target: "active"}},
				}),
			"active": primitives.NewStateConfig("active", primitives.Atomic),
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	m := NewMachine(config)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	var wg sync.WaitGroup
	const N = 50
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			m.Send(primitives.NewEvent("go", nil))
		}()
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond) // allow processing
	got := m.Current()
	if len(got) == 0 {
		t.Error("Current empty after concurrent sends")
	}
	// Expect active or idle, but consistent
}

func BenchmarkTransition(b *testing.B) {
	config := primitives.MachineConfig{
		ID:      "bench",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": primitives.NewStateConfig("idle", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"tick": {{Target: "active"}},
				}),
			"active": primitives.NewStateConfig("active", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"tick": {{Target: "idle"}},
				}),
		},
	}

	m := NewMachine(config)
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	defer m.Stop()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := m.Send(primitives.NewEvent("tick", nil)); err != nil {
			b.Fatal(err)
		}
	}
}
