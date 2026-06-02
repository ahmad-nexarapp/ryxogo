//go:build wasm

// recover_wasm.go — catches panics in Render() and shows a graceful error UI.
// Without this, any panic (nil pointer, index out of bounds etc) leaves the
// user with a completely blank screen and no indication of what went wrong.
package renderer

import (
	"fmt"
	"syscall/js"
)

// safeRender calls fn, catches any panic, and shows an error UI if needed.
// Returns false if a panic occurred.
func safeRender(rootEl js.Value, fn func()) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			ok = false
			showRenderError(rootEl, fmt.Sprintf("%v", rec))
		}
	}()
	fn()
	return true
}

func showRenderError(rootEl js.Value, msg string) {
	rootEl.Set("innerHTML", fmt.Sprintf(`
		<div style="
			padding: 2rem;
			font-family: monospace;
			background: #fef2f2;
			border: 1px solid #fecaca;
			border-radius: 8px;
			margin: 2rem;
			color: #991b1b;
		">
			<h2 style="margin:0 0 1rem;font-size:1rem">RyxoGo Render Error</h2>
			<pre style="
				background:#fff;
				padding:1rem;
				border-radius:4px;
				overflow:auto;
				font-size:0.8rem;
				color:#7f1d1d;
			">%s</pre>
			<p style="margin:1rem 0 0;font-size:0.8rem;color:#b91c1c">
				Check the browser console for the full stack trace.
			</p>
		</div>
	`, msg))
}
