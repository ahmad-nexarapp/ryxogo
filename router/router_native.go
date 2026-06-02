//go:build !wasm

// router_native.go — stubs for non-WASM environments
package router

func navigateTo(path string) {
	// In tests: no-op, state is managed by handlePath
}

func goBack() {}

func goForward() {}
