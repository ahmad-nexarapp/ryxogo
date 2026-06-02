//go:build wasm

// fetch_wasm.go — calls the browser's native fetch() API from Go WASM.
package http

import (
	"fmt"
	"syscall/js"
)

// doRequest uses the browser's fetch() API via JS interop
func doRequest(url, method string, headers map[string]string, body []byte) (*Response, error) {
	// Build fetch options
	opts := map[string]interface{}{
		"method": method,
	}

	// Headers
	jsHeaders := js.Global().Get("Object").New()
	for k, v := range headers {
		jsHeaders.Set(k, v)
	}
	opts["headers"] = jsHeaders

	// Body
	if len(body) > 0 {
		opts["body"] = string(body)
	}

	// Call browser fetch() — returns a Promise
	fetchPromise := js.Global().Call("fetch", url, opts)

	// Convert Promise to Go channel
	type result struct {
		status  int
		body    string
		headers map[string]string
		err     error
	}

	ch := make(chan result, 1)

	// .then() — success handler
	fetchPromise.Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			res := args[0]
			status := res.Get("status").Int()
			resHeaders := make(map[string]string)

			// Read response as text
			res.Call("text").Call("then",
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					ch <- result{
						status:  status,
						body:    args[0].String(),
						headers: resHeaders,
					}
					return nil
				}),
			)
			return nil
		}),
	).Call("catch",
		// .catch() — error handler
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			ch <- result{err: fmt.Errorf("fetch failed: %s", args[0].String())}
			return nil
		}),
	)

	// Wait for the promise to resolve
	r := <-ch
	if r.err != nil {
		return nil, r.err
	}

	return &Response{
		Status:  r.status,
		Body:    []byte(r.body),
		Headers: r.headers,
		OK:      r.status >= 200 && r.status < 300,
	}, nil
}
