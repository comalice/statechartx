package primitives

import (
	"fmt"
	"sync"
	"testing"
)

func TestContextBasic(t *testing.T) {
	ctx := NewContext()
	if _, ok := ctx.Get("nonexistent"); ok {
		t.Error("Get nonexistent should return false")
	}
	ctx.Set("key", 42)
	v, ok := ctx.Get("key")
	if !ok {
		t.Error("Get after Set should return true")
	}
	if vi, okk := v.(int); !okk || vi != 42 {
		t.Errorf("Get value mismatch: got %v (%T)", v, v)
	}
	ctx.Delete("key")
	_, ok = ctx.Get("key")
	if ok {
		t.Error("Get after Delete should return false")
	}
}

func TestContextConcurrentWritesAndReads(t *testing.T) {
	ctx := NewContext()
	const nWorkers = 50
	const nOpsPerWorker = 50
	var wg sync.WaitGroup
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < nOpsPerWorker; j++ {
				key := fmt.Sprintf("w%d_j%d", workerID, j)
				ctx.Set(key, j)
				v, has := ctx.Get(key)
				if !has || v.(int) != j {
					t.Errorf("Concurrent Set/Get mismatch for key %s: got %v", key, v)
				}
				if j%10 == 0 {
					ctx.Delete(key)
				}
			}
		}(i)
	}
	wg.Wait()
}
