// server.go — SSR HTTP server for RyxoGo.
// Renders pages to HTML on the server, then ships the WASM bundle
// which hydrates the static HTML (attaches event handlers).
package ssr

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// Meta holds per-page metadata for SEO and social sharing.
type Meta struct {
	Title       string
	Description string
	Image       string // Open Graph image URL
	URL         string
	Type        string // og:type — "website", "article" etc
	Keywords    string
}

// PageRenderer maps a path to a component + metadata.
type PageRenderer func(path string, params map[string]string) (core.Component, Meta)

// Server is an SSR server for RyxoGo apps.
type Server struct {
	routes    map[string]PageRenderer
	wasmPath  string // path to app.wasm on disk
	wasmExec  string // path to wasm_exec.js
	stylesURL string // URL to styles.css
	basePath  string
}

// NewServer creates an SSR server.
func NewServer() *Server {
	return &Server{
		routes:    make(map[string]PageRenderer),
		wasmPath:  "dist/app.wasm",
		wasmExec:  "dist/wasm_exec.js",
		stylesURL: "/styles.css",
	}
}

// Route registers a server-rendered route.
//
//	srv.Route("/", func(path string, p map[string]string) (core.Component, ssr.Meta) {
//	    return &HomePage{}, ssr.Meta{Title: "Home", Description: "Welcome"}
//	})
func (s *Server) Route(pattern string, renderer PageRenderer) *Server {
	s.routes[pattern] = renderer
	return s
}

// SetWASM configures the WASM bundle path.
func (s *Server) SetWASM(wasmPath, wasmExecPath string) *Server {
	s.wasmPath = wasmPath
	s.wasmExec = wasmExecPath
	return s
}

// Handler returns an http.Handler that serves SSR pages.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Serve WASM and static assets
	mux.HandleFunc("/app.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, s.wasmPath)
	})
	mux.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, s.wasmExec)
	})

	// All other routes get SSR
	mux.HandleFunc("/", s.handleSSR)

	return mux
}

// handleSSR renders the matching route to HTML.
func (s *Server) handleSSR(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Find matching route
	renderer, params, found := s.match(path)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		s.renderShell(w, "<div style='padding:2rem'><h1>404</h1><p>Page not found</p></div>",
			Meta{Title: "404 — Not Found"})
		return
	}

	comp, meta := renderer(path, params)
	if meta.URL == "" {
		meta.URL = path
	}

	// Render component to HTML
	contentHTML := RenderToString(comp)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.renderShell(w, contentHTML, meta)
}

// match finds the route for a path.
func (s *Server) match(path string) (PageRenderer, map[string]string, bool) {
	// Exact match first
	if r, ok := s.routes[path]; ok {
		return r, map[string]string{}, true
	}

	// Pattern match (e.g. /users/:id)
	for pattern, renderer := range s.routes {
		if params, ok := matchPattern(pattern, path); ok {
			return renderer, params, true
		}
	}
	return nil, nil, false
}

func matchPattern(pattern, path string) (map[string]string, bool) {
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")
	if len(pSegs) != len(pathSegs) {
		return nil, false
	}
	params := make(map[string]string)
	for i, seg := range pSegs {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = pathSegs[i]
		} else if seg != pathSegs[i] {
			return nil, false
		}
	}
	return params, true
}

// renderShell wraps content in the full HTML document with hydration.
func (s *Server) renderShell(w http.ResponseWriter, content string, meta Meta) {
	if meta.Type == "" {
		meta.Type = "website"
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
  <meta name="description" content="%s" />
  <meta name="keywords" content="%s" />

  <!-- Open Graph / social sharing -->
  <meta property="og:title" content="%s" />
  <meta property="og:description" content="%s" />
  <meta property="og:type" content="%s" />
  <meta property="og:url" content="%s" />
  <meta property="og:image" content="%s" />

  <!-- Twitter card -->
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="%s" />
  <meta name="twitter:description" content="%s" />

  <link rel="stylesheet" href="%s" />
  <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
</head>
<body>
  <!-- Server-rendered content — visible immediately, before WASM loads -->
  <div id="app" data-ssr="true">%s</div>

  <!-- WASM hydrates the static HTML above -->
  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('/app.wasm'), go.importObject)
      .then(r => go.run(r.instance))
      .catch(e => console.error('RyxoGo hydration error:', e));
  </script>
</body>
</html>`,
		meta.Title, meta.Description, meta.Keywords,
		meta.Title, meta.Description, meta.Type, meta.URL, meta.Image,
		meta.Title, meta.Description,
		s.stylesURL,
		content,
	)
}
