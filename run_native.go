//go:build !wasm

package ryxogo

// run is a no-op in non-WASM environments.
// The WASM version mounts the app to the real DOM.
func run(a *App) {
	// In tests or SSR: initialize router but don't touch DOM
	_ = a
}
