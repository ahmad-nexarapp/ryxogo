//go:build !wasm

// fetch_native.go — real net/http implementation for non-WASM environments.
// Used in tests and server-side rendering.
package http

import (
	"bytes"
	"fmt"
	ghttp "net/http"
	"io"
)

// doRequest performs the actual HTTP request using Go's net/http
func doRequest(url, method string, headers map[string]string, body []byte) (*Response, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := ghttp.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("ryxogo/http: failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &ghttp.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ryxogo/http: request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ryxogo/http: failed to read response: %w", err)
	}

	resHeaders := make(map[string]string)
	for k, v := range res.Header {
		if len(v) > 0 {
			resHeaders[k] = v[0]
		}
	}

	return &Response{
		Status:  res.StatusCode,
		Body:    resBody,
		Headers: resHeaders,
		OK:      res.StatusCode >= 200 && res.StatusCode < 300,
	}, nil
}
