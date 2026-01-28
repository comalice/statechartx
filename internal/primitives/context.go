// Package primitives provides foundational data structures for the statechart engine.
// All implementations use only the Go standard library for zero external dependencies.
// Context provides thread-safe key-value storage with RWMutex for concurrent access.
// Profiling may reveal opportunities to use sync.Map for lock-free reads.
//
//go:generate go test ./... -race
package primitives

import "sync"

// Context is a thread-safe key-value store using sync.Map for concurrent access.
// Lock-free reads/writes with good performance characteristics for contended access.
// Snapshot/Restore iterate the map for serialization.
type Context struct {
	data sync.Map
}

// NewContext creates a new Context with an empty map.
func NewContext() *Context {
	return &Context{}
}

// Get retrieves a value by key. Safe for concurrent reads.
func (c *Context) Get(key string) (any, bool) {
	return c.data.Load(key)
}

// Set stores a value by key. Exclusive write lock.
func (c *Context) Set(key string, val any) {
	c.data.Store(key, val)
}

// Delete removes a key-value pair. Exclusive write lock.
func (c *Context) Delete(key string) {
	c.data.Delete(key)
}

// Snapshot returns a serializable copy of the context data for persistence.
func (c *Context) Snapshot() map[string]any {
	snap := map[string]any{}
	c.data.Range(func(k, v any) bool {
		snap[k.(string)] = v
		return true
	})
	return snap
}

// Restore replaces the context data from a snapshot map.
func (c *Context) Restore(snap map[string]any) {
	c.data.Range(func(k, v any) bool {
		c.data.Delete(k)
		return true
	})
	for k, v := range snap {
		c.data.Store(k, v)
	}
}
