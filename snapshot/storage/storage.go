// Package storage keeps a global reference to the active simpleblob storage.
package storage

import (
	"sync"
	"time"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"
)

var (
	mu      sync.RWMutex
	storage simpleblob.Interface
	ready   = make(chan struct{}) // ready when closed
)

// SetGlobal sets the global storage backend.
func SetGlobal(st simpleblob.Interface) {
	if st == nil {
		panic("cannot set nil storage")
	}

	// Critical section includes the close(ready)
	mu.Lock()
	defer mu.Unlock()
	if storage == nil {
		// first time set
		close(ready)
	}
	storage = st
}

// GetGlobal retrieves the global storage set by SetGlobal.
// It blocks until one is available.
func GetGlobal() simpleblob.Interface {
	mu.RLock()
	st := storage
	mu.RUnlock()
	if st != nil {
		return st
	}

	// It looks like SetGlobal was not called yet
	wait()

	// Try again after wait. This time it must succeed.
	mu.RLock()
	st = storage
	mu.RUnlock()
	if st != nil {
		// Should never happen
		panic("Storage still nil after wait()")
	}
	return st
}

// IsReady reports if the storage is ready without blocking.
func IsReady() bool {
	mu.RLock()
	st := storage
	mu.RUnlock()
	return st != nil
}

// wait returns when the ready channel is closed, indicating the storage is
// available.
func wait() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ready:
			return
		case <-t.C:
			// This is here to prevent silent hangs when a command or test
			// forgets to perform the initialization.
			logrus.Warn("Still waiting for delta storage to be ready")
		}
	}
}
