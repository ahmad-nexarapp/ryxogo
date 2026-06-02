// Package router provides client-side routing for RyxoGo.
// Routes map URL patterns to Page components.
// File-based routing is handled by the CLI — developers
// don't configure routes manually.
package router

import (
	"strings"
)

// ---------------------------------------------------------
// Route definition
// ---------------------------------------------------------

// RouteHandler is a function that returns a Component for a route
type RouteHandler func(params map[string]string, query map[string]string) interface{}

// Route maps a URL pattern to a handler
type Route struct {
	Pattern  string       // "/users/:id", "/blog/:slug", "/"
	Handler  RouteHandler
	segments []string     // parsed pattern segments
}

// ---------------------------------------------------------
// Router
// ---------------------------------------------------------

// Router manages client-side navigation in RyxoGo
type Router struct {
	routes   []*Route
	current  string
	params   map[string]string
	query    map[string]string
	onChange func(route *Route, params, query map[string]string)
}

// New creates a new Router
func New() *Router {
	return &Router{
		params: make(map[string]string),
		query:  make(map[string]string),
	}
}

// Add registers a route pattern with a handler
// Pattern examples: "/", "/about", "/users/:id", "/blog/:year/:slug"
func (r *Router) Add(pattern string, handler RouteHandler) *Router {
	route := &Route{
		Pattern:  pattern,
		Handler:  handler,
		segments: parsePattern(pattern),
	}
	r.routes = append(r.routes, route)
	return r
}

// OnChange sets the callback for when the route changes.
// The renderer calls this to know when to re-render.
func (r *Router) OnChange(fn func(route *Route, params, query map[string]string)) {
	r.onChange = fn
}

// Navigate pushes a new URL and triggers a route change.
// Equivalent to React Router's navigate() or Vue Router's push().
func (r *Router) Navigate(path string) {
	// In WASM: updates browser history
	// In tests: updates internal state
	navigateTo(path)
	r.handlePath(path)
}

// Back navigates to the previous page
func (r *Router) Back() {
	goBack()
}

// Forward navigates forward
func (r *Router) Forward() {
	goForward()
}

// Current returns the current path
func (r *Router) Current() string {
	return r.current
}

// Params returns current route parameters
func (r *Router) Params() map[string]string {
	return r.params
}

// Query returns current query parameters
func (r *Router) Query() map[string]string {
	return r.query
}

// Param returns a single route parameter
func (r *Router) Param(key string) string {
	return r.params[key]
}

// handlePath matches a path against registered routes and fires onChange
func (r *Router) handlePath(path string) {
	// Split path and query string
	parts := strings.SplitN(path, "?", 2)
	pathname := parts[0]
	queryStr := ""
	if len(parts) > 1 {
		queryStr = parts[1]
	}

	r.current = pathname
	r.query = parseQuery(queryStr)

	// Match against registered routes
	for _, route := range r.routes {
		params, ok := matchRoute(route.segments, pathname)
		if ok {
			r.params = params
			if r.onChange != nil {
				r.onChange(route, params, r.query)
			}
			return
		}
	}

	// No route matched — 404
	r.params = make(map[string]string)
	if r.onChange != nil {
		r.onChange(nil, nil, r.query)
	}
}

// ---------------------------------------------------------
// Pattern matching
// ---------------------------------------------------------

func parsePattern(pattern string) []string {
	if pattern == "/" {
		return []string{"/"}
	}
	return strings.Split(strings.Trim(pattern, "/"), "/")
}

func matchRoute(patternSegs []string, path string) (map[string]string, bool) {
	if path == "/" && len(patternSegs) == 1 && patternSegs[0] == "/" {
		return map[string]string{}, true
	}

	pathSegs := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternSegs) != len(pathSegs) {
		return nil, false
	}

	params := make(map[string]string)
	for i, seg := range patternSegs {
		if strings.HasPrefix(seg, ":") {
			// Dynamic segment — capture as param
			params[seg[1:]] = pathSegs[i]
		} else if seg != pathSegs[i] {
			return nil, false
		}
	}

	return params, true
}

func parseQuery(queryStr string) map[string]string {
	result := make(map[string]string)
	if queryStr == "" {
		return result
	}
	for _, pair := range strings.Split(queryStr, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
