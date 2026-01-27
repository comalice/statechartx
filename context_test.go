package statechartx_test

import (
	"fmt"
	"sync"
	"testing"

	. "github.com/comalice/statechartx"
)

func TestContextBasic(t *testing.T) {
	ctx := NewContext()

	// Test Set/Get
	ctx.Set("key", "value")
	if got := ctx.Get("key"); got != "value" {
		t.Errorf("expected 'value', got %v", got)
	}

	// Test missing key returns nil
	if got := ctx.Get("missing"); got != nil {
		t.Errorf("expected nil for missing key, got %v", got)
	}

	// Test Delete
	ctx.Delete("key")
	if got := ctx.Get("key"); got != nil {
		t.Errorf("expected nil after delete, got %v", got)
	}
}

func TestContextTypes(t *testing.T) {
	ctx := NewContext()

	// Test different types
	ctx.Set("string", "value")
	ctx.Set("int", 42)
	ctx.Set("bool", true)
	ctx.Set("slice", []string{"a", "b", "c"})
	ctx.Set("map", map[string]int{"x": 1})

	if ctx.Get("string") != "value" {
		t.Error("string value mismatch")
	}
	if ctx.Get("int") != 42 {
		t.Error("int value mismatch")
	}
	if ctx.Get("bool") != true {
		t.Error("bool value mismatch")
	}
}

func TestContextConcurrency(t *testing.T) {
	ctx := NewContext()
	var wg sync.WaitGroup

	// 100 concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx.Set(fmt.Sprintf("key%d", id), id)
		}(i)
	}

	// 100 concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = ctx.Get(fmt.Sprintf("key%d", id))
		}(i)
	}

	// 50 concurrent deleters
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx.Delete(fmt.Sprintf("key%d", id))
		}(i)
	}

	wg.Wait()
	// No race conditions (run with -race flag)
}

func TestContextGetAll(t *testing.T) {
	ctx := NewContext()
	ctx.Set("a", 1)
	ctx.Set("b", 2)
	ctx.Set("c", 3)

	all := ctx.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}
	if all["a"] != 1 || all["b"] != 2 || all["c"] != 3 {
		t.Errorf("GetAll mismatch: %v", all)
	}

	// Mutation of snapshot doesn't affect original
	all["d"] = 4
	if ctx.Get("d") != nil {
		t.Error("GetAll should return defensive copy")
	}

	// Original still has 3 items
	all2 := ctx.GetAll()
	if len(all2) != 3 {
		t.Error("original context should be unchanged")
	}
}

func TestContextLoadAll(t *testing.T) {
	ctx := NewContext()
	ctx.Set("old", "value")
	ctx.Set("also_old", "data")

	newData := map[string]any{
		"new":     "data",
		"another": 123,
	}
	ctx.LoadAll(newData)

	// Old keys should be gone
	if ctx.Get("old") != nil {
		t.Error("LoadAll should replace, not merge - old key still exists")
	}
	if ctx.Get("also_old") != nil {
		t.Error("LoadAll should replace, not merge - also_old key still exists")
	}

	// New keys should exist
	if ctx.Get("new") != "data" {
		t.Error("LoadAll should set new data")
	}
	if ctx.Get("another") != 123 {
		t.Error("LoadAll should set all new data")
	}
}

func TestContextLoadAllNil(t *testing.T) {
	ctx := NewContext()
	ctx.Set("key", "value")

	// LoadAll with nil should clear everything
	ctx.LoadAll(nil)

	if ctx.Get("key") != nil {
		t.Error("LoadAll(nil) should clear context")
	}

	all := ctx.GetAll()
	if len(all) != 0 {
		t.Error("context should be empty after LoadAll(nil)")
	}
}

func TestRuntimeCtxAccessor(t *testing.T) {
	root := &State{ID: 1}
	machine, err := NewMachine(root)
	if err != nil {
		t.Fatal(err)
	}

	// Auto-created Context when ext is nil
	rt := NewRuntime(machine, nil)
	ctx := rt.Ctx()
	if ctx == nil {
		t.Fatal("expected auto-created Context")
	}

	// Should be able to use the context
	ctx.Set("test", "value")
	if ctx.Get("test") != "value" {
		t.Error("Context should work through Ctx() accessor")
	}

	// Custom ext (not a Context)
	customExt := map[string]string{"custom": "data"}
	rt2 := NewRuntime(machine, customExt)
	if rt2.Ctx() != nil {
		t.Error("Ctx() should return nil for non-Context ext")
	}

	// Explicitly created Context
	explicitCtx := NewContext()
	explicitCtx.Set("explicit", "value")
	rt3 := NewRuntime(machine, explicitCtx)
	if rt3.Ctx() != explicitCtx {
		t.Error("Ctx() should return the explicitly provided Context")
	}
	if rt3.Ctx().Get("explicit") != "value" {
		t.Error("explicit Context should preserve data")
	}
}

func TestContextOverwrite(t *testing.T) {
	ctx := NewContext()

	ctx.Set("key", "first")
	if ctx.Get("key") != "first" {
		t.Error("first set failed")
	}

	ctx.Set("key", "second")
	if ctx.Get("key") != "second" {
		t.Error("overwrite failed")
	}

	ctx.Set("key", 42)
	if ctx.Get("key") != 42 {
		t.Error("type change failed")
	}
}

func TestContextDeleteNonExistent(t *testing.T) {
	ctx := NewContext()

	// Deleting non-existent key should not panic
	ctx.Delete("nonexistent")

	// Should still be able to use context
	ctx.Set("key", "value")
	if ctx.Get("key") != "value" {
		t.Error("context should still work after deleting non-existent key")
	}
}
