// statechart.go - Minimal, composable, concurrent-ready hierarchical state machine (~520 LOC)
// Core features:
// - Hierarchical nesting with proper entry/exit order
// - Initial states and shallow history
// - Guarded transitions with actions
// - Thread-safe event dispatch
// - Designed for explicit composition: easy to run multiple instances concurrently via channels
// - No built-in parallel regions — parallelism achieved through composition + goroutines

// Package statechartx provides a minimal, composable, concurrent-ready hierarchical state machine (~520 LOC).\n// Core features:\n// - Hierarchical nesting with proper entry/exit order\n// - Initial states and shallow history\n// - Guarded transitions with actions\n// - Thread-safe event dispatch\n// - Designed for explicit composition: easy to run multiple instances concurrently via channels\n// - No built-in parallel regions — parallelism achieved through composition + goroutines\npackage statechart
package statechartx

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Event is any type. Use comparable types for events to avoid runtime panics on == checks.
type Event any

// Action is called on entry/exit or during transitions
type Action func(ctx context.Context, event Event, from, to StateID, ext any)

// Guard returns true if the transition should be taken
type Guard func(ctx context.Context, event Event, from, to StateID, ext any) bool

// StateID uniquely identifies a state
type StateID string

// State defines a node in the hierarchy
type State struct {
	ID          StateID
	Parent      *State
	Children    map[StateID]*State
	Initial     *State // default substate on entry
	History     *State // shallow history: last active child
	OnEntry     Action
	OnExit      Action
	Transitions []*Transition
}

// Transition defines an outgoing edge
type Transition struct {
	Event  Event
	Target StateID
	Guard  Guard
	Action Action
}

// Runtime executes one state machine instance
type Runtime struct {
	root       *State
	current    map[*State]struct{} // active states
	ext        any                 // extended state / user context
	mu         sync.RWMutex
	running    bool
	eventQueue []Event // Internal events (synchronous FIFO)
	processing bool    // Detect recursion/internal calls
}

// NewRuntime creates a new executable state machine
func NewRuntime(root *State, extendedContext any) *Runtime {
	return &Runtime{
		root:       root,
		current:    make(map[*State]struct{}),
		ext:        extendedContext,
		eventQueue: []Event{},
		processing: false,
	}
}

// Start enters the initial configuration
func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("already running")
	}
	r.running = true
	r.current = make(map[*State]struct{})
	r.mu.Unlock()
	if err := r.enterInitial(ctx, r.root); err != nil {
		return err
	}
	r.processMicrosteps(ctx)
	return nil
}

// Stop exits all active states
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	// Snapshot active states
	var active []*State
	for s := range r.current {
		active = append(active, s)
	}
	r.current = make(map[*State]struct{})
	r.running = false
	r.mu.Unlock()
	// Exit unlocked
	for _, s := range active {
		r.exitState(ctx, s)
	}
	return nil
}

// SendEvent dispatches an event (thread-safe)
func (r *Runtime) SendEvent(ctx context.Context, event Event) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return fmt.Errorf("not running")
	}
	if r.processing {
		r.eventQueue = append(r.eventQueue, event)
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	enabled := r.findEnabledTransition(event)
	if enabled == nil {
		return nil
	}

	source := enabled.source
	targetState := enabled.targetState

	// Find LCA for proper exit/entry
	lca := r.findLCA(source, targetState)

	// States to exit (source up to but not including LCA) - tree traversal, lock-free
	var exitSet []*State
	cur := source
	for cur != nil && cur != lca {
		exitSet = append(exitSet, cur)
		cur = cur.Parent
	}

	// Exit bottom-up
	sort.Slice(exitSet, func(i, j int) bool {
		return len(r.ancestors(exitSet[i])) > len(r.ancestors(exitSet[j]))
	})
	for _, s := range exitSet {
		r.exitState(ctx, s)
	}

	// Transition action unlocked
	if enabled.Action != nil {
		enabled.Action(ctx, event, source.ID, enabled.Target, r.ext)
	}

	// Enter target configuration
	if err := r.enterState(ctx, targetState); err != nil {
		return err
	}
	r.processMicrosteps(ctx)
	return nil
}

// Raise enqueues an event at the front of the event queue.
func (r *Runtime) Raise(ctx context.Context, event Event) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return fmt.Errorf("not running")
	}
	// Prepend to queue for internal priority
	r.eventQueue = append([]Event{event}, r.eventQueue...)
	r.mu.Unlock()
	return nil
}

// IsInState returns true if the state (or any descendant) is active
func (r *Runtime) IsInState(id StateID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	target := r.findStateByID(r.root, id)
	if target == nil {
		return false
	}
	if _, ok := r.current[target]; ok {
		return true
	}
	for s := range r.current {
		if r.isDescendant(s, target) {
			return true
		}
	}
	return false
}

func (r *Runtime) processMicrosteps(ctx context.Context) {
	for {
		r.mu.Lock()
		if len(r.eventQueue) == 0 {
			r.mu.Unlock()
			break
		}
		ev := r.eventQueue[0]
		r.eventQueue = r.eventQueue[1:]
		r.mu.Unlock()
		r.processing = true
		if err := r.processSingleEvent(ctx, ev); err != nil {
			// Ignore for conformance
		}
		r.processing = false
	}

	// Process eventless / completion transitions until stable.
	for {
		enabled := r.findEnabledTransition(nil) // check only eventless transitions
		if enabled == nil {
			break
		}
		_ = r.processSingleEvent(ctx, nil)
	}
}

func (r *Runtime) processSingleEvent(ctx context.Context, ev Event) error {
	enabled := r.findEnabledTransition(ev)
	if enabled == nil {
		return nil
	}
	source := enabled.source
	targetState := enabled.targetState

	// Find LCA for proper exit/entry
	lca := r.findLCA(source, targetState)

	// States to exit (source up to but not including LCA)
	var exitSet []*State
	cur := source
	for cur != nil && cur != lca {
		exitSet = append(exitSet, cur)
		cur = cur.Parent
	}

	// Exit bottom-up
	sort.Slice(exitSet, func(i, j int) bool {
		return len(r.ancestors(exitSet[i])) > len(r.ancestors(exitSet[j]))
	})
	for _, s := range exitSet {
		r.exitState(ctx, s)
	}

	// Transition action unlocked
	if enabled.Action != nil {
		enabled.Action(ctx, ev, source.ID, enabled.Target, r.ext)
	}

	// Enter target configuration
	if err := r.enterState(ctx, targetState); err != nil {
		return err
	}
	r.processMicrosteps(ctx)
	return nil
}

// RunAsActor runs the machine in its own goroutine, driven by an input channel
// Perfect for concurrent composition (orthogonal "regions")
func (r *Runtime) RunAsActor(parentCtx context.Context, input <-chan Event) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	if err := r.Start(ctx); err != nil {
		return // log or handle in real code
	}

	defer r.Stop(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-input:
			if !ok {
				return
			}
			_ = r.SendEvent(ctx, ev) // fire-and-forget; errors ignored for simplicity
		}
	}
}

// ------------------------ Private Helpers ------------------------

type enabledTransition struct {
	source      *State
	targetState *State
	Target      StateID
	Action      Action
}

func (r *Runtime) findEnabledTransition(event Event) *enabledTransition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	active := r.activeStatesOrdered() // deepest first

	for i := len(active) - 1; i >= 0; i-- {
		s := active[i]
		for _, t := range s.Transitions {
			if t.Event == event || (t.Event == nil && event == nil) { // Match normal event or eventless transition
				if t.Guard == nil || t.Guard(context.Background(), event, s.ID, t.Target, r.ext) {
					target := r.findStateByID(r.root, t.Target)
					if target != nil {
						return &enabledTransition{
							source:      s,
							targetState: target,
							Target:      t.Target,
							Action:      t.Action,
						}
					}
				}
			}
		}
	}
	return nil
}

func (r *Runtime) activeStatesOrdered() []*State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*State
	for s := range r.current {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return len(r.ancestors(list[i])) > len(r.ancestors(list[j]))
	})
	return list
}

func (r *Runtime) ancestors(s *State) []*State {
	var chain []*State
	for cur := s; cur != nil; cur = cur.Parent {
		chain = append(chain, cur)
	}
	return chain
}

func (r *Runtime) findLCA(a, b *State) *State {
	ancA := r.ancestors(a)
	ancB := r.ancestors(b)

	// Reverse to root-first
	for i, j := 0, len(ancA)-1; i < j; i, j = i+1, j-1 {
		ancA[i], ancA[j] = ancA[j], ancA[i]
	}
	for i, j := 0, len(ancB)-1; i < j; i, j = i+1, j-1 {
		ancB[i], ancB[j] = ancB[j], ancB[i]
	}

	lca := ancA[0]
	minLen := len(ancA)
	if len(ancB) < minLen {
		minLen = len(ancB)
	}
	for i := 0; i < minLen; i++ {
		if ancA[i] == ancB[i] {
			lca = ancA[i]
		} else {
			break
		}
	}
	return lca
}

func (r *Runtime) enterInitial(ctx context.Context, composite *State) error {
	if composite.Initial != nil {
		return r.enterState(ctx, composite.Initial)
	}
	for _, child := range composite.Children {
		return r.enterState(ctx, child)
	}
	return nil
}

func (r *Runtime) enterState(ctx context.Context, s *State) error {
	r.mu.Lock()
	r.current[s] = struct{}{}
	r.mu.Unlock()
	if s.OnEntry != nil {
		s.OnEntry(ctx, nil, "", s.ID, r.ext)
	}
	if len(s.Children) > 0 {
		if s.History != nil {
			err := r.enterState(ctx, s.History)
			if err != nil {
				return err
			}
		} else {
			err := r.enterInitial(ctx, s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runtime) exitState(ctx context.Context, s *State) error {
	// Collect children under RLock for snapshot
	var children []*State
	r.mu.RLock()
	for child := range r.current {
		if child.Parent == s {
			children = append(children, child)
		}
	}
	r.mu.RUnlock()
	// Exit children recursively (unlocked)
	for _, child := range children {
		r.exitState(ctx, child)
	}
	// Delete from current under lock
	r.mu.Lock()
	delete(r.current, s)
	r.mu.Unlock()
	// OnExit unlocked (safe for callbacks)
	if s.OnExit != nil {
		s.OnExit(ctx, nil, s.ID, "", r.ext)
	}
	// History under lock
	r.mu.Lock()
	if s.Parent != nil {
		s.Parent.History = s
	}
	r.mu.Unlock()
	return nil
}

func (r *Runtime) findStateByID(cur *State, id StateID) *State {
	if cur == nil {
		return nil
	}
	if cur.ID == id {
		return cur
	}
	for _, child := range cur.Children {
		if found := r.findStateByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func (r *Runtime) isDescendant(child, ancestor *State) bool {
	for cur := child.Parent; cur != nil; cur = cur.Parent {
		if cur == ancestor {
			return true
		}
	}
	return false
}
