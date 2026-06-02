// Package router provides client-side routing for RyxoGo.
package router

import "strings"

// RouteHandler returns a Component for a route
type RouteHandler func(params map[string]string, query map[string]string) interface{}

// Route maps a URL pattern to a handler
type Route struct {
	Pattern  string
	Handler  RouteHandler
	segments []string
}

// Router manages client-side navigation
type Router struct {
	routes   []*Route
	current  string
	params   map[string]string
	query    map[string]string
	basePath string // FIX: base path for subpath deploys e.g. "/app"
	onChange func(route *Route, params, query map[string]string)
}

// New creates a new Router
func New() *Router {
	return &Router{
		params: make(map[string]string),
		query:  make(map[string]string),
	}
}

// SetBasePath sets the base path for subpath deploys.
// e.g. SetBasePath("/app") makes /app/about match route /about
func (r *Router) SetBasePath(base string) {
	r.basePath = strings.TrimRight(base, "/")
}

// Add registers a route
func (r *Router) Add(pattern string, handler RouteHandler) *Router {
	route := &Route{
		Pattern:  pattern,
		Handler:  handler,
		segments: parsePattern(pattern),
	}
	r.routes = append(r.routes, route)
	return r
}

// OnChange sets the re-render callback
func (r *Router) OnChange(fn func(route *Route, params, query map[string]string)) {
	r.onChange = fn
}

// Navigate pushes a new URL
func (r *Router) Navigate(path string) {
	navigateTo(r.basePath + path)
	r.handlePath(path)
}

// Back navigates to previous page
func (r *Router) Back() { goBack() }

// Forward navigates forward
func (r *Router) Forward() { goForward() }

// Current returns the current path (without basePath)
func (r *Router) Current() string { return r.current }

// Params returns current route parameters
func (r *Router) Params() map[string]string { return r.params }

// Query returns current query parameters
func (r *Router) Query() map[string]string { return r.query }

// Param returns a single route parameter
func (r *Router) Param(key string) string { return r.params[key] }

// handlePath strips basePath, then matches against routes
func (r *Router) handlePath(rawPath string) {
	// Strip basePath prefix
	path := rawPath
	if r.basePath != "" && strings.HasPrefix(rawPath, r.basePath) {
		path = rawPath[len(r.basePath):]
		if path == "" {
			path = "/"
		}
	}

	// Split path and query
	parts := strings.SplitN(path, "?", 2)
	pathname := parts[0]
	if pathname == "" {
		pathname = "/"
	}
	queryStr := ""
	if len(parts) > 1 {
		queryStr = parts[1]
	}

	r.current = pathname
	r.query = parseQuery(queryStr)

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

	// 404
	r.params = make(map[string]string)
	if r.onChange != nil {
		r.onChange(nil, nil, r.query)
	}
}

// CurrentFullPath returns the raw browser path for popstate
func (r *Router) HandleBrowserPath(rawPath string) {
	r.handlePath(rawPath)
}

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
