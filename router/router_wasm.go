//go:build wasm

// router_wasm.go — connects RyxoGo router to browser history API
package router

import "syscall/js"

// navigateTo pushes a new entry to browser history
func navigateTo(path string) {
	js.Global().Get("history").Call("pushState", nil, "", path)
}

// goBack calls browser history.back()
func goBack() {
	js.Global().Get("history").Call("back")
}

// goForward calls browser history.forward()
func goForward() {
	js.Global().Get("history").Call("forward")
}
