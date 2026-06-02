// Package http provides the fetch() API for RyxoGo.
// Wraps browser fetch via WASM JS interop.
package http

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------------------------------------------------------
// Request builder — fetch("/api/users").Method("POST").Do()
// ---------------------------------------------------------

// Request is a chainable HTTP request builder.
// fetch() returns one of these.
type Request struct {
	url     string
	method  string
	headers map[string]string
	body    interface{}
	params  map[string]string
}

// Fetch creates a new HTTP request to the given URL.
// This is the primary API — developers call fetch("/api/users")
func Fetch(url string) *Request {
	return &Request{
		url:     url,
		method:  "GET",
		headers: make(map[string]string),
		params:  make(map[string]string),
	}
}

// Method sets the HTTP method (GET, POST, PUT, DELETE, PATCH)
func (r *Request) Method(method string) *Request {
	r.method = strings.ToUpper(method)
	return r
}

// Header adds a single HTTP header
func (r *Request) Header(key, value string) *Request {
	r.headers[key] = value
	return r
}

// Headers sets multiple headers at once
func (r *Request) Headers(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

// Body sets the request body — accepts any JSON-serializable value
func (r *Request) Body(body interface{}) *Request {
	r.body = body
	r.headers["Content-Type"] = "application/json"
	return r
}

// Param adds a URL query parameter
func (r *Request) Param(key, value string) *Request {
	r.params[key] = value
	return r
}

// Params adds multiple URL query parameters
func (r *Request) Params(params map[string]string) *Request {
	for k, v := range params {
		r.params[k] = v
	}
	return r
}

// Bearer sets the Authorization: Bearer <token> header
func (r *Request) Bearer(token string) *Request {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// Auth is an alias for Bearer
func (r *Request) Auth(token string) *Request {
	return r.Bearer(token)
}

// Post is a shorthand to set method to POST with a body
func (r *Request) Post(body interface{}) *Request {
	return r.Method("POST").Body(body)
}

// Put is a shorthand to set method to PUT with a body
func (r *Request) Put(body interface{}) *Request {
	return r.Method("PUT").Body(body)
}

// Delete is a shorthand for DELETE
func (r *Request) Delete() *Request {
	return r.Method("DELETE")
}

// ---------------------------------------------------------
// Response — what Do() returns
// ---------------------------------------------------------

// Response wraps an HTTP response
type Response struct {
	Status  int
	Body    []byte
	Headers map[string]string
	OK      bool // true if status 200-299
}

// JSON deserializes the response body into the given value
func (res *Response) JSON(v interface{}) error {
	return json.Unmarshal(res.Body, v)
}

// Text returns the response body as a string
func (res *Response) Text() string {
	return string(res.Body)
}

// ---------------------------------------------------------
// Do — executes the request
// In WASM: calls browser fetch via JS interop
// In tests: uses Go's net/http
// ---------------------------------------------------------

// Do executes the HTTP request and returns a Response.
// In the browser (WASM build) this calls the browser's fetch() API.
// In tests and server-side rendering this uses Go's net/http.
func (r *Request) Do() (*Response, error) {
	// Build URL with query params
	url := r.url
	if len(r.params) > 0 {
		parts := make([]string, 0, len(r.params))
		for k, v := range r.params {
			parts = append(parts, k+"="+v)
		}
		url += "?" + strings.Join(parts, "&")
	}

	// Serialize body
	var bodyBytes []byte
	if r.body != nil {
		var err error
		bodyBytes, err = json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("ryxogo/http: failed to serialize body: %w", err)
		}
	}

	// In WASM builds, this delegates to the JS bridge (js/fetch_wasm.go)
	// In non-WASM builds, this uses Go's net/http (http/fetch_native.go)
	return doRequest(url, r.method, r.headers, bodyBytes)
}

// DoJSON executes the request and automatically decodes JSON into v
func (r *Request) DoJSON(v interface{}) error {
	res, err := r.Do()
	if err != nil {
		return err
	}
	if !res.OK {
		return fmt.Errorf("ryxogo/http: request failed with status %d", res.Status)
	}
	return res.JSON(v)
}

// ---------------------------------------------------------
// Convenience functions — the simplest API
// ---------------------------------------------------------

// Get performs a GET request and decodes JSON response
func Get[T any](url string) (T, error) {
	var result T
	err := Fetch(url).DoJSON(&result)
	return result, err
}

// Post performs a POST request with JSON body and decodes response
func Post[T any](url string, body interface{}) (T, error) {
	var result T
	err := Fetch(url).Method("POST").Body(body).DoJSON(&result)
	return result, err
}

// Put performs a PUT request with JSON body
func Put[T any](url string, body interface{}) (T, error) {
	var result T
	err := Fetch(url).Method("PUT").Body(body).DoJSON(&result)
	return result, err
}

// Del performs a DELETE request
func Del(url string) error {
	_, err := Fetch(url).Method("DELETE").Do()
	return err
}
