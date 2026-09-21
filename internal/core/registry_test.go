package core

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRegistryConcurrentRegister(t *testing.T) {
	reg := NewRegistry(15 * time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("inst-%d", i)
			reg.Register(id, "container-"+id, "host-1", "1.2.3.4:1000", 10)
		}(i)
	}
	wg.Wait()

	all := reg.All()
	if len(all) != 50 {
		t.Fatalf("expected 50 instances, got %d", len(all))
	}
}

func TestRegistryConcurrentHeartbeatAndRemove(t *testing.T) {
	reg := NewRegistry(15 * time.Second)
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("inst-%d", i)
		reg.Register(id, "container-"+id, "host-1", "1.2.3.4:1000", 10)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("inst-%d", i)
		wg.Add(2)
		go func(id string) {
			defer wg.Done()
			reg.Heartbeat(id, 5)
		}(id)
		go func(id string) {
			defer wg.Done()
			_ = reg.All()
		}(id)
	}
	wg.Wait()
}

func TestRegistryConcurrentMixedOperations(t *testing.T) {
	reg := NewRegistry(15 * time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(4)
		id := fmt.Sprintf("inst-%d", i)

		go func(id string) {
			defer wg.Done()
			reg.Register(id, "container-"+id, "host-1", "1.2.3.4:1000", 10)
		}(id)
		go func(id string) {
			defer wg.Done()
			reg.Heartbeat(id, 3)
		}(id)
		go func() {
			defer wg.Done()
			_ = reg.All()
		}()
		go func(id string) {
			defer wg.Done()
			reg.Remove(id)
		}(id)
	}
	wg.Wait()
}
