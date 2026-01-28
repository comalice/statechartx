package statechartx

import "sync"

// Context provides thread-safe storage for extended state.
// Use with NewRuntime(machine, nil) for auto-creation,
// or NewRuntime(machine, customExt) to preserve custom ext.
type Context struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewContext creates an empty context.
func NewContext() *Context {
	return &Context{
		data: make(map[string]any),
	}
}

// Get retrieves a value by key. Returns nil if the key does not exist.
func (c *Context) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

// Set stores a value by key.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Delete removes a key from the context.
func (c *Context) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// GetAll returns a snapshot copy of all data for serialization.
// The returned map is a defensive copy and modifications will not affect the context.
func (c *Context) GetAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]any, len(c.data))
	for k, v := range c.data {
		snapshot[k] = v
	}
	return snapshot
}

// LoadAll atomically replaces all data in the context.
// This is useful for deserialization.
func (c *Context) LoadAll(data map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
}
