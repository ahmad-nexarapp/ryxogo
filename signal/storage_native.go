//go:build !wasm

// storage_native.go — no-op storage for non-WASM builds.
package signal

import "sync"

// In-memory fallback for tests and SSR
var memStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func init() {
	memStorage.data = make(map[string]string)

	StorageRead = func(key string) (string, bool) {
		memStorage.mu.RLock()
		defer memStorage.mu.RUnlock()
		v, ok := memStorage.data[key]
		return v, ok
	}

	StorageWrite = func(key, value string) {
		memStorage.mu.Lock()
		memStorage.data[key] = value
		memStorage.mu.Unlock()
	}

	StorageDel = func(key string) {
		memStorage.mu.Lock()
		delete(memStorage.data, key)
		memStorage.mu.Unlock()
	}
}
