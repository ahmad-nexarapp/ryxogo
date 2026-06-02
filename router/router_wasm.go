//go:build wasm

// router_wasm.go — browser history API integration.
// F6 FIX: uses replaceState when navigating to same path.
package router

import "syscall/js"

func navigateTo(path string) {
	current := js.Global().Get("location").Get("pathname").String()
	if current == path {
		// F6 FIX: replaceState instead of pushState to avoid duplicate history entries
		js.Global().Get("history").Call("replaceState", nil, "", path)
	} else {
		js.Global().Get("history").Call("pushState", nil, "", path)
	}
}

func goBack() {
	js.Global().Get("history").Call("back")
}

func goForward() {
	js.Global().Get("history").Call("forward")
}
