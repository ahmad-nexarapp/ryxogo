//go:build wasm

// storage_wasm.go — wires localStorage to PersistSignal in the browser.
package signal

import "syscall/js"

func init() {
	ls := js.Global().Get("localStorage")

	StorageRead = func(key string) (string, bool) {
		val := ls.Call("getItem", key)
		if val.IsNull() || val.IsUndefined() {
			return "", false
		}
		return val.String(), true
	}

	StorageWrite = func(key, value string) {
		ls.Call("setItem", key, value)
	}

	StorageDel = func(key string) {
		ls.Call("removeItem", key)
	}
}
