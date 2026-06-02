// Package mcp implements the RyxoGo MCP (Model Context Protocol) server.
// This lets AI tools like Cursor, Claude Code, and Codex understand
// your RyxoGo app structure in real time.
//
// When you run `rxgo mcp serve`, this starts on port 7777.
// AI tools connect to it and can:
//   - List all components, pages, stores
//   - Read component source and types
//   - Understand routes and their params
//   - Generate new components that fit perfectly
//   - Validate their output before writing files
package mcp

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------
// App Schema — what the MCP server exposes to AI tools
// ---------------------------------------------------------

// AppSchema is a complete machine-readable description of a RyxoGo app.
// AI tools use this to understand the project before generating code.
type AppSchema struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Framework   string              `json:"framework"`
	GeneratedAt time.Time           `json:"generatedAt"`
	Routes      []RouteSchema       `json:"routes"`
	Components  []ComponentSchema   `json:"components"`
	Pages       []PageSchema        `json:"pages"`
	Stores      []StoreSchema       `json:"stores"`
	Types       []TypeSchema        `json:"types"`
	Conventions ConventionSchema    `json:"conventions"`
}

// RouteSchema describes a single route
type RouteSchema struct {
	Pattern    string   `json:"pattern"`
	Page       string   `json:"page"`
	File       string   `json:"file"`
	Params     []string `json:"params"`     // [:id, :slug]
	Protected  bool     `json:"protected"`
	Methods    []string `json:"methods"`
}

// ComponentSchema describes a reusable component
type ComponentSchema struct {
	Name        string      `json:"name"`
	File        string      `json:"file"`
	Props       []PropSchema `json:"props"`
	Signals     []SignalSchema `json:"signals"`
	Description string      `json:"description"`
	Example     string      `json:"example"`
}

// PageSchema describes a page component
type PageSchema struct {
	Name    string      `json:"name"`
	File    string      `json:"file"`
	Route   string      `json:"route"`
	Signals []SignalSchema `json:"signals"`
	Fetches []FetchSchema `json:"fetches"`
}

// StoreSchema describes a global store
type StoreSchema struct {
	Name    string       `json:"name"`
	File    string       `json:"file"`
	Fields  []PropSchema `json:"fields"`
	Actions []string     `json:"actions"`
}

// TypeSchema describes a shared Go type
type TypeSchema struct {
	Name   string       `json:"name"`
	File   string       `json:"file"`
	Fields []PropSchema `json:"fields"`
	Kind   string       `json:"kind"` // struct, interface, type alias
}

// PropSchema describes a single field or prop
type PropSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

// SignalSchema describes a signal declared in Setup()
type SignalSchema struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // Signal, Computed, Async
	ValType string `json:"valType"` // int, string, []User, etc
}

// FetchSchema describes a fetch() call
type FetchSchema struct {
	Signal   string `json:"signal"`
	URL      string `json:"url"`
	Method   string `json:"method"`
	RespType string `json:"respType"`
}

// ConventionSchema describes the project's coding conventions
// so AI tools generate consistent code
type ConventionSchema struct {
	StylingFramework string   `json:"stylingFramework"` // tailwind, plain css
	FileNaming       string   `json:"fileNaming"`       // snake_case, camelCase
	ComponentStyle   string   `json:"componentStyle"`   // struct-based
	StatePattern     string   `json:"statePattern"`     // signals
	APIBaseURL       string   `json:"apiBaseURL"`
	ExistingClasses  []string `json:"existingClasses"`  // common Tailwind classes used
}

// ---------------------------------------------------------
// Scanner — reads the project and builds the schema
// ---------------------------------------------------------

// Scanner scans a RyxoGo project directory and builds an AppSchema
type Scanner struct {
	root   string
	fset   *token.FileSet
	schema *AppSchema
}

// NewScanner creates a scanner for the given project root
func NewScanner(root string) *Scanner {
	return &Scanner{
		root: root,
		fset: token.NewFileSet(),
		schema: &AppSchema{
			Framework:   "ryxogo",
			Version:     "1.0.0",
			GeneratedAt: time.Now(),
			Conventions: ConventionSchema{
				ComponentStyle: "struct-based",
				StatePattern:   "signals",
			},
		},
	}
}

// Scan walks the project and builds the full schema
func (s *Scanner) Scan() (*AppSchema, error) {
	// Read project name from go.mod
	if name, err := s.readModuleName(); err == nil {
		s.schema.Name = name
	}

	// Scan pages/
	if err := s.scanPages(); err != nil {
		return nil, err
	}

	// Scan components/
	if err := s.scanComponents(); err != nil {
		return nil, err
	}

	// Scan stores/
	if err := s.scanStores(); err != nil {
		return nil, err
	}

	// Scan types/
	if err := s.scanTypes(); err != nil {
		return nil, err
	}

	// Detect styling framework
	s.detectStyling()

	return s.schema, nil
}

// readModuleName reads the module name from go.mod
func (s *Scanner) readModuleName() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			parts := strings.Split(strings.TrimPrefix(line, "module "), "/")
			return parts[len(parts)-1], nil
		}
	}
	return "", fmt.Errorf("module name not found")
}

// scanPages scans the pages/ directory
func (s *Scanner) scanPages() error {
	pagesDir := filepath.Join(s.root, "pages")
	if _, err := os.Stat(pagesDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		return s.parsePageFile(path)
	})
}

// scanComponents scans the components/ directory
func (s *Scanner) scanComponents() error {
	compDir := filepath.Join(s.root, "components")
	if _, err := os.Stat(compDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(compDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		return s.parseComponentFile(path)
	})
}

// scanStores scans the stores/ directory
func (s *Scanner) scanStores() error {
	storesDir := filepath.Join(s.root, "stores")
	if _, err := os.Stat(storesDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(storesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		return s.parseStoreFile(path)
	})
}

// scanTypes scans the types/ directory
func (s *Scanner) scanTypes() error {
	typesDir := filepath.Join(s.root, "types")
	if _, err := os.Stat(typesDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(typesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		return s.parseTypesFile(path)
	})
}

// parsePageFile parses a single page file and extracts schema info
func (s *Scanner) parsePageFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := parser.ParseFile(s.fset, path, src, parser.ParseComments)
	if err != nil {
		return nil // skip files with syntax errors
	}

	rel, _ := filepath.Rel(s.root, path)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Check if it embeds rx.Page
			if !embedsPage(structType) {
				continue
			}

			page := PageSchema{
				Name:  typeSpec.Name.Name,
				File:  rel,
				Route: fileToRoute(rel),
			}

			// Extract signals from struct fields
			page.Signals = extractSignals(structType)

			s.schema.Pages = append(s.schema.Pages, page)

			// Also add to routes
			route := RouteSchema{
				Pattern: page.Route,
				Page:    page.Name,
				File:    rel,
				Params:  extractRouteParams(page.Route),
			}
			s.schema.Routes = append(s.schema.Routes, route)
		}
	}

	return nil
}

// parseComponentFile parses a component file
func (s *Scanner) parseComponentFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := parser.ParseFile(s.fset, path, src, parser.ParseComments)
	if err != nil {
		return nil
	}

	rel, _ := filepath.Rel(s.root, path)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Look for Props struct
			if strings.HasSuffix(typeSpec.Name.Name, "Props") {
				continue
			}

			comp := ComponentSchema{
				Name:    typeSpec.Name.Name,
				File:    rel,
				Signals: extractSignals(structType),
				Props:   extractProps(f, typeSpec.Name.Name),
			}

			// Extract doc comment
			if genDecl.Doc != nil {
				comp.Description = strings.TrimSpace(genDecl.Doc.Text())
			}

			s.schema.Components = append(s.schema.Components, comp)
		}
	}

	return nil
}

// parseStoreFile parses a store file
func (s *Scanner) parseStoreFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := parser.ParseFile(s.fset, path, src, parser.ParseComments)
	if err != nil {
		return nil
	}

	rel, _ := filepath.Rel(s.root, path)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if !strings.HasSuffix(typeSpec.Name.Name, "Store") {
				continue
			}

			store := StoreSchema{
				Name:   typeSpec.Name.Name,
				File:   rel,
				Fields: extractStructFields(structType),
			}

			s.schema.Stores = append(s.schema.Stores, store)
		}
	}

	return nil
}

// parseTypesFile parses shared types
func (s *Scanner) parseTypesFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := parser.ParseFile(s.fset, path, src, parser.ParseComments)
	if err != nil {
		return nil
	}

	rel, _ := filepath.Rel(s.root, path)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			t := TypeSchema{
				Name: typeSpec.Name.Name,
				File: rel,
				Kind: "struct",
			}

			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				t.Fields = extractStructFields(structType)
			}

			s.schema.Types = append(s.schema.Types, t)
		}
	}

	_ = f
	return nil
}

// detectStyling detects which CSS framework is being used
func (s *Scanner) detectStyling() {
	// Check for tailwind config
	if _, err := os.Stat(filepath.Join(s.root, "tailwind.config.js")); err == nil {
		s.schema.Conventions.StylingFramework = "tailwind"
		return
	}
	s.schema.Conventions.StylingFramework = "plain"
}

// ---------------------------------------------------------
// AST helpers
// ---------------------------------------------------------

func embedsPage(s *ast.StructType) bool {
	for _, field := range s.Fields.List {
		if len(field.Names) == 0 { // embedded field
			if sel, ok := field.Type.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Page" || sel.Sel.Name == "Base" {
					return true
				}
			}
			if ident, ok := field.Type.(*ast.Ident); ok {
				if ident.Name == "Page" || ident.Name == "Base" {
					return true
				}
			}
		}
	}
	return false
}

func extractSignals(s *ast.StructType) []SignalSchema {
	var signals []SignalSchema
	for _, field := range s.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		fieldType := fmt.Sprintf("%v", field.Type)
		if strings.Contains(fieldType, "Signal") ||
			strings.Contains(fieldType, "Computed") ||
			strings.Contains(fieldType, "Async") {
			for _, name := range field.Names {
				sig := SignalSchema{
					Name: name.Name,
					Type: signalKind(fieldType),
				}
				signals = append(signals, sig)
			}
		}
	}
	return signals
}

func signalKind(t string) string {
	if strings.Contains(t, "Async") {
		return "Async"
	}
	if strings.Contains(t, "Computed") {
		return "Computed"
	}
	return "Signal"
}

func extractStructFields(s *ast.StructType) []PropSchema {
	var props []PropSchema
	for _, field := range s.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			props = append(props, PropSchema{
				Name: name.Name,
				Type: fmt.Sprintf("%v", field.Type),
			})
		}
	}
	return props
}

func extractProps(f *ast.File, componentName string) []PropSchema {
	propsName := componentName + "Props"
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != propsName {
				continue
			}
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				return extractStructFields(structType)
			}
		}
	}
	return nil
}

func fileToRoute(file string) string {
	// pages/index.go → /
	// pages/about.go → /about
	// pages/users/[id].go → /users/:id
	// pages/blog/[slug].go → /blog/:slug

	file = strings.TrimPrefix(file, "pages/")
	file = strings.TrimSuffix(file, ".go")

	if file == "index" {
		return "/"
	}

	parts := strings.Split(file, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			parts[i] = ":" + p[1:len(p)-1]
		}
		if p == "index" {
			parts[i] = ""
		}
	}

	route := "/" + strings.Join(parts, "/")
	return strings.ReplaceAll(route, "//", "/")
}

func extractRouteParams(route string) []string {
	var params []string
	for _, part := range strings.Split(route, "/") {
		if strings.HasPrefix(part, ":") {
			params = append(params, part[1:])
		}
	}
	return params
}

// ---------------------------------------------------------
// MCP Server — exposes tools to AI clients
// ---------------------------------------------------------

// Server is the RyxoGo MCP server
type Server struct {
	root    string
	scanner *Scanner
	schema  *AppSchema
}

// NewServer creates a new MCP server for the project at root
func NewServer(root string) *Server {
	return &Server{
		root:    root,
		scanner: NewScanner(root),
	}
}

// Tools returns the list of MCP tools this server exposes
func (s *Server) Tools() []Tool {
	return []Tool{
		{
			Name:        "get_app_schema",
			Description: "Returns the complete RyxoGo app schema — all components, pages, routes, stores, and types. Call this first before generating any code.",
			InputSchema: map[string]interface{}{},
		},
		{
			Name:        "get_component",
			Description: "Returns the full source code and schema for a specific component.",
			InputSchema: map[string]interface{}{
				"name": "string — the component name e.g. ProductCard",
			},
		},
		{
			Name:        "get_page",
			Description: "Returns the full source code and schema for a specific page.",
			InputSchema: map[string]interface{}{
				"name": "string — the page name e.g. DashboardPage",
			},
		},
		{
			Name:        "list_routes",
			Description: "Returns all registered routes and their page components.",
			InputSchema: map[string]interface{}{},
		},
		{
			Name:        "list_types",
			Description: "Returns all shared types — use these when generating code to avoid duplicates.",
			InputSchema: map[string]interface{}{},
		},
		{
			Name:        "validate_component",
			Description: "Validates a generated component for correctness before writing to disk.",
			InputSchema: map[string]interface{}{
				"code": "string — the Go component source code to validate",
			},
		},
		{
			Name:        "get_conventions",
			Description: "Returns the project coding conventions — styling framework, naming, patterns.",
			InputSchema: map[string]interface{}{},
		},
		{
			Name:        "get_ryxogo_docs",
			Description: "Returns RyxoGo framework documentation for a specific topic.",
			InputSchema: map[string]interface{}{
				"topic": "string — one of: signals, components, routing, fetching, stores, styling",
			},
		},
		{
			Name:        "generate_component_scaffold",
			Description: "Generates a starter scaffold for a new component based on a description.",
			InputSchema: map[string]interface{}{
				"name":        "string — component name",
				"description": "string — what the component does",
				"hasProps":    "bool — whether it needs props",
				"hasFetch":    "bool — whether it fetches data",
			},
		},
	}
}

// Tool describes an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Call handles an MCP tool call and returns the result
func (s *Server) Call(tool string, args map[string]interface{}) (interface{}, error) {
	// Refresh schema on every call so it's always up to date
	schema, err := s.scanner.Scan()
	if err != nil {
		return nil, err
	}
	s.schema = schema

	switch tool {
	case "get_app_schema":
		return s.schema, nil

	case "get_component":
		name, _ := args["name"].(string)
		return s.getComponentSource(name)

	case "get_page":
		name, _ := args["name"].(string)
		return s.getPageSource(name)

	case "list_routes":
		return s.schema.Routes, nil

	case "list_types":
		return s.schema.Types, nil

	case "validate_component":
		code, _ := args["code"].(string)
		return s.validateComponent(code)

	case "get_conventions":
		return s.schema.Conventions, nil

	case "get_ryxogo_docs":
		topic, _ := args["topic"].(string)
		return getRyxoGoDocs(topic), nil

	case "generate_component_scaffold":
		return s.generateScaffold(args)

	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}

// getComponentSource reads the source file for a component
func (s *Server) getComponentSource(name string) (map[string]interface{}, error) {
	for _, c := range s.schema.Components {
		if c.Name == name {
			src, err := os.ReadFile(filepath.Join(s.root, c.File))
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"schema": c,
				"source": string(src),
			}, nil
		}
	}
	return nil, fmt.Errorf("component %q not found", name)
}

// getPageSource reads the source file for a page
func (s *Server) getPageSource(name string) (map[string]interface{}, error) {
	for _, p := range s.schema.Pages {
		if p.Name == name {
			src, err := os.ReadFile(filepath.Join(s.root, p.File))
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"schema": p,
				"source": string(src),
			}, nil
		}
	}
	return nil, fmt.Errorf("page %q not found", name)
}

// validateComponent checks if generated Go code is valid
func (s *Server) validateComponent(code string) (map[string]interface{}, error) {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	if err != nil {
		return map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		}, nil
	}
	return map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	}, nil
}

// generateScaffold creates a starter component based on description
func (s *Server) generateScaffold(args map[string]interface{}) (string, error) {
	name, _ := args["name"].(string)
	hasProps, _ := args["hasProps"].(bool)
	hasFetch, _ := args["hasFetch"].(bool)

	var sb strings.Builder
	sb.WriteString("package components\n\n")
	sb.WriteString("import rx \"github.com/ahmad-nexarapp/ryxogo\"\n\n")

	if hasProps {
		sb.WriteString(fmt.Sprintf("type %sProps struct {\n\t// TODO: define props\n}\n\n", name))
	}

	sb.WriteString(fmt.Sprintf("type %s struct {\n\trx.Base\n", name))
	if hasProps {
		sb.WriteString(fmt.Sprintf("\tProps %sProps\n", name))
	}
	if hasFetch {
		sb.WriteString("\t// TODO: add async signals\n")
	}
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("func (c *%s) Setup() {\n", name))
	if hasFetch {
		sb.WriteString("\t// c.data = rx.Async(func() (YourType, error) {\n")
		sb.WriteString("\t//     return rx.Get[YourType](\"/api/...\")\n")
		sb.WriteString("\t// })\n")
	}
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("func (c *%s) Render() *rx.Node {\n", name))
	sb.WriteString("\treturn rx.Div(rx.Props{},\n")
	sb.WriteString("\t\t// TODO: render content\n")
	sb.WriteString("\t)\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}

// getRyxoGoDocs returns framework documentation for a topic
func getRyxoGoDocs(topic string) string {
	docs := map[string]string{
		"signals": `
## RyxoGo Signals

Signals are reactive variables. When their value changes, any component
reading them automatically re-renders.

### Creating signals (in Setup())
  count := rx.Use(0)          // Signal[int]
  name  := rx.Use("")         // Signal[string]
  items := rx.Use([]Item{})   // Signal[[]Item]

### Reading
  count.Val()    // returns current value, subscribes component

### Writing (triggers re-render)
  count.Set(5)
  count.Update(func(v int) int { return v + 1 })

### Computed (auto-updates when dependencies change)
  total := rx.Computed(func() float64 {
      return price.Val() * float64(qty.Val())
  })

### Watch (side effects)
  rx.Watch(func() {
      fmt.Println("count is:", count.Val())
      // auto-tracks count — no dep array
  })
`,

		"components": `
## RyxoGo Components

Every component is a Go struct embedding rx.Base or rx.Page.

### Structure
  type MyComponent struct {
      rx.Base
      Props MyComponentProps
      // signals declared here
      count *rx.Signal[int]
  }

  // Setup() — called once, declare signals here
  func (c *MyComponent) Setup() {
      c.count = rx.Use(0)
  }

  // Render() — called on every render, return the UI
  func (c *MyComponent) Render() *rx.Node {
      return rx.Div(rx.Props{Class: "..."},
          rx.Text(strconv.Itoa(c.count.Val())),
      )
  }

### Lifecycle
  OnMount()   — runs when component enters the DOM
  OnUnmount() — runs when component leaves the DOM
  OnUpdate()  — runs when props change

### Props
  type ButtonProps struct {
      Label   string
      OnClick func()
      Variant string
  }
`,

		"fetching": `
## RyxoGo Data Fetching

Use rx.Async() for any async data. It handles loading/error/success automatically.

### Basic fetch
  p.users = rx.Async(func() ([]User, error) {
      return rx.Get[[]User]("/api/users")
  })

### In Render()
  if p.users.IsLoading() { return Spinner{}.Render() }
  if p.users.IsError()   { return ErrorCard{}.Render() }
  users := p.users.Data()

### HTTP client
  // GET
  rx.Get[User]("/api/users/123")

  // POST
  rx.Post[Response]("/api/users", map[string]any{"name": "Alice"})

  // Chainable
  rx.Fetch("/api/users").
      Bearer(token).
      Param("page", "2").
      DoJSON(&result)

### Refetch
  p.users.Refetch() // re-runs the fetch function
`,

		"routing": `
## RyxoGo Routing

File-based: pages/about.go → /about, pages/users/[id].go → /users/:id

### Manual route registration (main.go)
  app.Route("/", &HomePage{})
  app.Route("/users/:id", &UserPage{})

### Reading route params (in page)
  p.Param("id")       // route param
  p.Query("search")   // query string

### Navigation
  rx.Navigate("/dashboard")
  rx.Back()
`,

		"stores": `
## RyxoGo Stores (Global State)

Stores hold state shared across multiple components.

### Define (stores/auth.go)
  type AuthStore struct {
      User  *User
      Token string
  }
  var Auth = rx.NewStore(&AuthStore{})

### Read (any component)
  user := rx.GetStore(Auth).User

### Write
  rx.UpdateStore(Auth, func(s *AuthStore) {
      s.User = loggedInUser
  })
`,

		"styling": `
## RyxoGo Styling

Works with any CSS framework. Tailwind is recommended.

### Tailwind classes
  rx.Div(rx.Props{Class: "flex items-center gap-4 p-8"}, ...)

### Conditional classes
  class := "px-4 py-2 rounded "
  if active { class += "bg-blue-600 text-white" } else { class += "bg-gray-100" }
  rx.Button(rx.Props{Class: class}, ...)

### Inline styles
  rx.Div(rx.Props{
      Style: map[string]string{
          "background": "#4f46e5",
          "padding":    "1rem",
      },
  }, ...)
`,
	}

	if doc, ok := docs[topic]; ok {
		return doc
	}

	// Return all docs if topic not found
	var all strings.Builder
	for _, v := range docs {
		all.WriteString(v)
		all.WriteString("\n---\n")
	}
	return all.String()
}

// ---------------------------------------------------------
// JSON output helpers
// ---------------------------------------------------------

// SchemaToJSON serializes the app schema to JSON
func SchemaToJSON(schema *AppSchema) ([]byte, error) {
	return json.MarshalIndent(schema, "", "  ")
}
