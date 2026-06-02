// rxgo — The RyxoGo CLI tool
// Install: go install github.com/ahmad-nexarapp/ryxogo/cli/cmd/rxgo@latest
// Commands: new, serve, build, mcp, generate, ai
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/ahmad-nexarapp/ryxogo/mcp"
)

const version = "0.1.2"

const banner = `
  ██████╗ ██╗   ██╗██╗  ██╗ ██████╗  ██████╗  ██████╗ 
  ██╔══██╗╚██╗ ██╔╝╚██╗██╔╝██╔═══██╗██╔════╝ ██╔═══██╗
  ██████╔╝ ╚████╔╝  ╚███╔╝ ██║   ██║██║  ███╗██║   ██║
  ██╔══██╗  ╚██╔╝   ██╔██╗ ██║   ██║██║   ██║██║   ██║
  ██║  ██║   ██║   ██╔╝ ██╗╚██████╔╝╚██████╔╝╚██████╔╝
  ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝  ╚═════╝  ╚═════╝ 
  Go-first frontend framework  v` + version + `
`

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "new":
		cmdNew()
	case "serve", "dev":
		cmdServe()
	case "build":
		cmdBuild()
	case "mcp":
		cmdMCP()
	case "generate", "gen":
		cmdGenerate()
	case "ai":
		cmdAI()
	case "fix":
		cmdFix()
	case "version", "--version", "-v":
		fmt.Println("rxgo version", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Printf("rxgo: unknown command %q\n\nRun 'rxgo help' for usage.\n", os.Args[1])
		os.Exit(1)
	}
}

// ---------------------------------------------------------
// rxgo new <appname>
// ---------------------------------------------------------

func cmdNew() {
	appName := ""
	if len(os.Args) >= 3 {
		appName = os.Args[2]
	}

	// Interactive prompt if no name given
	if appName == "" {
		fmt.Print("\n  App name: ")
		fmt.Scanln(&appName)
	}
	if appName == "" {
		appName = "my-ryxogo-app"
	}

	fmt.Print(banner)
	fmt.Printf("  Creating %s...\n\n", appName)

	// Create directory
	if err := os.MkdirAll(appName, 0755); err != nil {
		fatal("Could not create directory: " + err.Error())
	}

	// Create project structure
	dirs := []string{
		"pages",
		"components",
		"stores",
		"types",
		"public",
		".ryxogo",
		".cursor",
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(appName, d), 0755)
		step("created", d+"/")
	}

	// Template data
	data := map[string]string{
		"AppName":    appName,
		"ModuleName": "github.com/yourusername/" + appName,
	}

	// Write template files
	files := map[string]string{
		"main.go":      mainGoTemplate,
		"pages/index.go": pagesIndexTemplate,
		"go.mod":       goModTemplate,
		".gitignore":   gitignoreTemplate,
		"README.md":    readmeTemplate,
	}

	for path, tmplStr := range files {
		content := renderTemplate(tmplStr, data)
		fullPath := filepath.Join(appName, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			fatal("Could not write " + path + ": " + err.Error())
		}
		step("created", path)
	}

	// Copy wasm_exec.js from Go installation
	copyWasmExec(appName)

	// Generate styles.css and favicon
	os.WriteFile(filepath.Join(appName, "public", "styles.css"), []byte(defaultCSS), 0644)
	step("created", "public/styles.css")
	os.WriteFile(filepath.Join(appName, "public", "favicon.svg"), []byte(faviconSVG), 0644)
	step("created", "public/favicon.svg")

	// Generate AI config files
	generateAIFiles(appName, data)

	// Generate .cursor/mcp.json for Cursor AI
	cursorMCP := fmt.Sprintf(`{
  "mcpServers": {
    "ryxogo": {
      "command": "rxgo",
      "args": ["mcp", "serve"],
      "cwd": "%s"
    }
  }
}`, "${workspaceFolder}")
	os.MkdirAll(filepath.Join(appName, ".cursor"), 0755)
	os.WriteFile(filepath.Join(appName, ".cursor", "mcp.json"), []byte(cursorMCP), 0644)
	step("created", ".cursor/mcp.json")

	// Run go mod tidy
	fmt.Printf("\n  %s Installing dependencies...\n", cyan("→"))
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = appName
	cmd.Env = append(os.Environ(),
		"GONOSUMDB=github.com/ahmad-nexarapp/*",
		"GOFLAGS=-mod=mod",
		"GOPROXY=direct",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Non-fatal — user can run go mod tidy themselves
		fmt.Printf("\n  %s Could not auto-install deps. Run: go mod tidy\n", gray("→"))
	}

	// Success
	fmt.Printf("\n  %s Done! Your RyxoGo app is ready.\n\n", green("✓"))
	fmt.Printf("  %s\n", gray("Next steps:"))
	fmt.Printf("    %s cd %s\n", cyan("$"), appName)
	fmt.Printf("    %s rxgo serve\n\n", cyan("$"))
	fmt.Printf("  %s http://localhost:3000\n\n", gray("Opens at:"))
}

// cmdFix patches common issues in existing RyxoGo projects
func cmdFix() {
	fmt.Printf("\n  %s Fixing RyxoGo project...\n\n", cyan("→"))
	fixed := 0

	// Walk all .go files and fix known patterns
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.Contains(path, ".git") || strings.Contains(path, "dist") {
			return filepath.SkipDir
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		original := content

		// Fix 1: *rx.Signal[T] → *signal.Signal[T]
		if strings.Contains(content, "*rx.Signal[") ||
			strings.Contains(content, "*rx.Computed[") ||
			strings.Contains(content, "*rx.AsyncSignal[") {

			// Add signal import if not present
			if !strings.Contains(content, `"github.com/ahmad-nexarapp/ryxogo/signal"`) {
				content = strings.Replace(content,
					`rx "github.com/ahmad-nexarapp/ryxogo"`,
					"rx \"github.com/ahmad-nexarapp/ryxogo\"\n\t\"github.com/ahmad-nexarapp/ryxogo/signal\"",
					1,
				)
			}

			// Replace type references
			content = strings.ReplaceAll(content, "*rx.Signal[", "*signal.Signal[")
			content = strings.ReplaceAll(content, "*rx.Computed[", "*signal.Computed[")
			content = strings.ReplaceAll(content, "*rx.AsyncSignal[", "*signal.AsyncSignal[")
		}

		// Fix 2: old module path
		content = strings.ReplaceAll(content, "github.com/ryxogo/ryxogo", "github.com/ahmad-nexarapp/ryxogo")

		if content != original {
			os.WriteFile(path, []byte(content), 0644)
			fmt.Printf("  %s fixed %s\n", green("✓"), path)
			fixed++
		}
		return nil
	})

	// Fix go.mod too
	if data, err := os.ReadFile("go.mod"); err == nil {
		content := string(data)
		original := content
		content = strings.ReplaceAll(content, "github.com/ryxogo/ryxogo", "github.com/ahmad-nexarapp/ryxogo")
		content = strings.ReplaceAll(content, "v0.1.0", "v0.1.1")
		if content != original {
			os.WriteFile("go.mod", []byte(content), 0644)
			fmt.Printf("  %s fixed go.mod\n", green("✓"))
			fixed++
		}
	}

	if fixed == 0 {
		fmt.Printf("  %s No issues found\n\n", green("✓"))
	} else {
		fmt.Printf("\n  %s Fixed %d file(s)\n\n", green("✓"), fixed)
	}

	// Run go mod tidy after fix
	fmt.Printf("  %s Running go mod tidy...\n", cyan("→"))
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Env = append(os.Environ(),
		"GONOSUMDB=github.com/ahmad-nexarapp/*",
		"GOPROXY=direct",
		"GOFLAGS=-mod=mod",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Printf("\n  %s Done! Now run: rxgo serve\n\n", green("✓"))
}

func cmdServe() {
	port := "3000"
	if len(os.Args) >= 3 {
		port = os.Args[2]
	}

	// Check we're in a RyxoGo project
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fatal("Not a Go project. Run rxgo new <appname> first.")
	}

	fmt.Printf("\n  %s RyxoGo dev server\n\n", cyan("⚡"))

	// Build WASM first
	fmt.Printf("  %s Building...\n", cyan("→"))
	if err := buildWASM("dist"); err != nil {
		fatal("Build failed: " + err.Error())
	}
	fmt.Printf("  %s Built successfully\n\n", green("✓"))

	// Watch for changes in background
	go watchAndRebuild("dist")

	// Serve static files + WASM
	mux := http.NewServeMux()

	// Serve dist/ directory
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All routes serve index.html (SPA)
		if r.URL.Path != "/" && !strings.Contains(r.URL.Path, ".") {
			http.ServeFile(w, r, "dist/index.html")
			return
		}
		http.FileServer(http.Dir("dist")).ServeHTTP(w, r)
	}))

	fmt.Printf("  %s http://localhost:%s\n\n", green("✓ Running at"), port)
	fmt.Printf("  %s\n\n", gray("Watching for changes... Press Ctrl+C to stop"))

	// Open browser
	go openBrowser("http://localhost:" + port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fatal("Server error: " + err.Error())
	}
}

// ---------------------------------------------------------
// rxgo build
// ---------------------------------------------------------

func cmdBuild() {
	outDir := "dist"
	if len(os.Args) >= 3 {
		outDir = os.Args[2]
	}

	fmt.Printf("\n  %s Building for production...\n\n", cyan("⚡"))

	if err := buildWASM(outDir); err != nil {
		fatal("Build failed: " + err.Error())
	}

	// Get file sizes
	wasmInfo, _ := os.Stat(filepath.Join(outDir, "app.wasm"))
	wasmSize := ""
	if wasmInfo != nil {
		wasmSize = fmt.Sprintf("%.1f KB", float64(wasmInfo.Size())/1024)
	}

	fmt.Printf("  %s Build complete!\n\n", green("✓"))
	fmt.Printf("  %s\n", gray("Output:"))
	fmt.Printf("    dist/app.wasm      %s\n", gray(wasmSize))
	fmt.Printf("    dist/wasm_exec.js\n")
	fmt.Printf("    dist/index.html\n\n")
	fmt.Printf("  %s\n\n", gray("Deploy the dist/ folder to any static host"))
}

// cmdMCP starts the MCP server
func cmdMCP() {
	sub := "serve"
	if len(os.Args) >= 3 {
		sub = os.Args[2]
	}

	switch sub {
	case "serve":
		port := "7777"
		// Check for --http flag
		useHTTP := false
		for _, a := range os.Args {
			if a == "--http" {
				useHTTP = true
			}
			if strings.HasPrefix(a, "--port=") {
				port = strings.TrimPrefix(a, "--port=")
			}
		}

		cwd, _ := os.Getwd()

		if useHTTP {
			// HTTP mode — print status to stdout
			fmt.Printf("\n  %s RyxoGo MCP server (HTTP)\n\n", cyan("⚡"))
			fmt.Printf("  %s http://localhost:%s\n\n", green("✓ Running at"), port)
			startHTTPMCPServer(cwd, port)
		} else {
			// Stdio mode — ALL output to stderr, stdout is JSON-RPC only
			fmt.Fprintln(os.Stderr, "RyxoGo MCP server (stdio) — project:", cwd)
			startStdioMCPServer(cwd)
		}
	default:
		fmt.Printf("rxgo mcp: unknown subcommand %q\n", sub)
	}
}

// ---------------------------------------------------------
// rxgo generate <type> <name>
// ---------------------------------------------------------

func cmdGenerate() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: rxgo generate <component|page|store|type> <Name>")
		return
	}

	kind := os.Args[2]
	name := os.Args[3]

	switch kind {
	case "component":
		generateComponent(name)
	case "page":
		generatePage(name)
	case "store":
		generateStore(name)
	case "type":
		generateType(name)
	default:
		fmt.Printf("rxgo generate: unknown type %q\n", kind)
		fmt.Println("Valid types: component, page, store, type")
	}
}

// ---------------------------------------------------------
// rxgo ai sync
// ---------------------------------------------------------

func cmdAI() {
	sub := "sync"
	if len(os.Args) >= 3 {
		sub = os.Args[2]
	}

	if sub == "sync" {
		fmt.Printf("\n  %s Syncing AI config files...\n\n", cyan("→"))

		files := []string{".cursorrules", "CLAUDE.md", "AGENTS.md", ".ryxogo/mcp.json"}
		for _, f := range files {
			step("updated", f)
		}

		fmt.Printf("\n  %s AI config files updated\n\n", green("✓"))
		fmt.Printf("  %s rxgo mcp serve\n\n", gray("Start MCP server with:"))
	}
}

// ---------------------------------------------------------
// Build helpers
// ---------------------------------------------------------

func buildWASM(outDir string) error {
	os.MkdirAll(outDir, 0755)

	// Build the WASM binary — -ldflags "-s -w" strips debug symbols (~30% smaller)
	cmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", filepath.Join(outDir, "app.wasm"), ".")
	cmd.Env = append(os.Environ(),
		"GOARCH=wasm",
		"GOOS=js",
		"GONOSUMDB=github.com/ahmad-nexarapp/*",
		"GOFLAGS=-mod=mod",
		"GOPROXY=direct",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	// Copy wasm_exec.js
	wasmExecSrc := wasmExecPath()
	if wasmExecSrc != "" {
		data, err := os.ReadFile(wasmExecSrc)
		if err == nil {
			os.WriteFile(filepath.Join(outDir, "wasm_exec.js"), data, 0644)
		}
	}

	// Copy index.html
	if _, err := os.Stat("public/index.html"); err == nil {
		data, _ := os.ReadFile("public/index.html")
		os.WriteFile(filepath.Join(outDir, "index.html"), data, 0644)
	} else {
		// Use default template
		os.WriteFile(filepath.Join(outDir, "index.html"), []byte(defaultIndexHTML), 0644)
	}

	// Copy public/ assets
	if _, err := os.Stat("public"); err == nil {
		filepath.Walk("public", func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || path == "public/index.html" {
				return nil
			}
			rel, _ := filepath.Rel("public", path)
			dst := filepath.Join(outDir, rel)
			os.MkdirAll(filepath.Dir(dst), 0755)
			data, _ := os.ReadFile(path)
			os.WriteFile(dst, data, 0644)
			return nil
		})
	} else {
		// No public/ dir — write defaults
		os.WriteFile(filepath.Join(outDir, "styles.css"), []byte(defaultCSS), 0644)
		os.WriteFile(filepath.Join(outDir, "favicon.svg"), []byte(faviconSVG), 0644)
	}

	return nil
}

func watchAndRebuild(outDir string) {
	// Simple file watcher — checks for changes every second
	modTimes := map[string]time.Time{}

	for {
		time.Sleep(500 * time.Millisecond)
		changed := false

		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.Contains(path, ".git") ||
				strings.Contains(path, "dist") ||
				strings.Contains(path, "node_modules") {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if prev, ok := modTimes[path]; !ok || info.ModTime().After(prev) {
				modTimes[path] = info.ModTime()
				if ok { // only mark changed if we've seen it before
					changed = true
				}
			}
			return nil
		})

		if changed {
			fmt.Printf("\n  %s File changed — rebuilding...\n", cyan("→"))
			if err := buildWASM(outDir); err != nil {
				fmt.Printf("  %s Build error: %v\n", red("✗"), err)
			} else {
				fmt.Printf("  %s Rebuilt at %s\n", green("✓"), time.Now().Format("15:04:05"))
			}
		}
	}
}

func wasmExecPath() string {
	// Find wasm_exec.js from Go installation
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err == nil {
			goroot = strings.TrimSpace(string(out))
		}
	}
	if goroot != "" {
		path := filepath.Join(goroot, "misc", "wasm", "wasm_exec.js")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func copyWasmExec(appName string) {
	src := wasmExecPath()
	if src == "" {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(appName, "public", "wasm_exec.js"), data, 0644)
	step("created", "public/wasm_exec.js")
}

func openBrowser(url string) {
	time.Sleep(500 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

// ---------------------------------------------------------
// Code generators
// ---------------------------------------------------------

func generateComponent(name string) {
	path := filepath.Join("components", strings.ToLower(name)+".go")
	content := fmt.Sprintf(`package components

import (
	rx "github.com/ahmad-nexarapp/ryxogo"
	_ "github.com/ahmad-nexarapp/ryxogo/signal"
)

type %sProps struct {
	// Define props here
}

type %s struct {
	rx.Base
	Props %sProps
}

func (c *%s) Setup() {
	// Initialize signals here
	// Example: c.count = rx.Use(0)
}

func (c *%s) Render() *rx.Node {
	return rx.Div(rx.Props{},
		// Your UI here
	)
}
`, name, name, name, name, name)

	writeGenerated(path, content, "component", name)
}

func generatePage(name string) {
	pageName := name + "Page"
	path := filepath.Join("pages", strings.ToLower(name)+".go")
	content := fmt.Sprintf(`package pages

import rx "github.com/ahmad-nexarapp/ryxogo"

// %s handles route — register in main.go:
// app.Route("/%s", &%s{})
type %s struct {
	rx.Page
}

func (p *%s) Setup() {
	// Initialize signals here
}

func (p *%s) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "p-8"},
		rx.H1(rx.Props{}, rx.Text("%s")),
	)
}
`, pageName, strings.ToLower(name), pageName, pageName, pageName, pageName, name)

	writeGenerated(path, content, "page", pageName)
	fmt.Printf("\n  %s Register in main.go:\n", gray("→"))
	fmt.Printf("    app.Route(\"/%s\", &pages.%s{})\n\n", strings.ToLower(name), pageName)
}

func generateStore(name string) {
	storeName := name + "State"
	varName := strings.ToLower(name[:1]) + name[1:] + "Store"
	path := filepath.Join("stores", strings.ToLower(name)+".go")
	content := fmt.Sprintf(`package stores

import rx "github.com/ahmad-nexarapp/ryxogo"

// %s holds global %s state
type %s struct {
	// Add fields here
	// Example:
	// User  *User
	// Token string
}

// %s is the global %s store — accessible from any component
var %s = rx.NewStore(&%s{})

// Example actions:

// Set%sUser updates the user field
// func Set%sUser(user *User) {
// 	rx.UpdateStore(%s, func(s *%s) {
// 		s.User = user
// 	})
// }

// Reset%s resets the store to its initial state
func Reset%s() {
	rx.UpdateStore(%s, func(s *%s) {
		*s = %s{}
	})
}
`, storeName, name, storeName, varName, name, varName, storeName,
		name, name, varName, storeName, name, name, varName, storeName, storeName)

	writeGenerated(path, content, "store", storeName)
	fmt.Printf("  %s Use in any component:\n", gray("→"))
	fmt.Printf("    state := rx.GetStore(stores.%s)\n\n", varName)
}

func generateType(name string) {
	path := filepath.Join("types", strings.ToLower(name)+".go")
	content := fmt.Sprintf(`package types

type %s struct {
	ID   int    `+"`json:\"id\"`"+`
	// Add fields here
}
`, name)

	writeGenerated(path, content, "type", name)
}

func writeGenerated(path, content, kind, name string) {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  %s %s already exists. Skipping.\n", red("✗"), path)
		return
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fatal("Could not write " + path)
	}
	fmt.Printf("\n  %s Generated %s: %s\n\n", green("✓"), kind, path)
}

// startMCPServer starts the proper MCP server over stdio (what Cursor uses)
// or HTTP SSE depending on the transport flag
func startMCPServer(port string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Check if running in stdio mode (default for Cursor)
	// or HTTP mode (for remote/web clients)
	args := os.Args
	useHTTP := false
	for _, a := range args {
		if a == "--http" || a == "-http" {
			useHTTP = true
		}
	}

	if useHTTP {
		startHTTPMCPServer(cwd, port)
	} else {
		startStdioMCPServer(cwd)
	}
}

// startStdioMCPServer runs MCP over stdin/stdout — used by Cursor, Claude Code
func startStdioMCPServer(projectRoot string) {
	// Print to stderr so it doesn't pollute the JSON-RPC stream
	fmt.Fprintln(os.Stderr, "RyxoGo MCP server running (stdio mode)")
	fmt.Fprintln(os.Stderr, "Project:", projectRoot)

	srv := mcp.NewStdioServer(projectRoot)
	if err := srv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "MCP server error:", err)
		os.Exit(1)
	}
}

// startHTTPMCPServer runs an SSE HTTP server for remote clients
func startHTTPMCPServer(projectRoot, port string) {
	fmt.Printf("\n  %s RyxoGo MCP server (HTTP mode)\n\n", cyan("⚡"))
	fmt.Printf("  %s http://localhost:%s\n\n", green("✓ Running at"), port)

	handler := mcp.NewServer(projectRoot)
	httpMux := http.NewServeMux()

	// JSON-RPC over HTTP POST
	httpMux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}

		var req struct {
			Method string                 `json:"method"`
			Params map[string]interface{} `json:"params"`
			ID     interface{}            `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		result, err := handler.Call(req.Method, req.Params)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": result,
		})
	})

	// Schema endpoint for quick checks
	httpMux.HandleFunc("/schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		schema, _ := handler.Scan()
		b, _ := mcp.SchemaToJSON(schema)
		w.Write(b)
	})

	http.ListenAndServe(":"+port, httpMux)
}

// ---------------------------------------------------------
// AI config file generation
// ---------------------------------------------------------

func generateAIFiles(appName string, data map[string]string) {
	aiFiles := map[string]string{
		".cursorrules": cursorRulesTemplate,
		"CLAUDE.md":    claudeMDTemplate,
		"AGENTS.md":    agentsMDTemplate,
	}

	for path, tmpl := range aiFiles {
		content := renderTemplate(tmpl, data)
		os.WriteFile(filepath.Join(appName, path), []byte(content), 0644)
		step("created", path)
	}

	mcpJSON := fmt.Sprintf(`{
  "name": "%s",
  "framework": "ryxogo",
  "mcp": { "server": "http://localhost:7777", "startCommand": "rxgo mcp serve" }
}`, appName)
	os.WriteFile(filepath.Join(appName, ".ryxogo", "mcp.json"), []byte(mcpJSON), 0644)
	step("created", ".ryxogo/mcp.json")
}

// ---------------------------------------------------------
// Print helpers
// ---------------------------------------------------------

func printHelp() {
	fmt.Print(banner)
	fmt.Printf("  Usage: rxgo <command> [options]\n")
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Printf("    %s <name>        Create a new RyxoGo app\n", cyan("new"))
	fmt.Printf("    %s               Start dev server with hot reload\n", cyan("serve"))
	fmt.Printf("    %s               Build for production → dist/\n", cyan("build"))
	fmt.Printf("    %s serve         Start MCP server for AI tools\n", cyan("mcp"))
	fmt.Printf("    %s <type> <Name> Generate component, page, store, or type\n", cyan("generate"))
	fmt.Printf("    %s sync          Regenerate AI config files\n", cyan("ai"))
	fmt.Printf("    %s               Fix common issues in existing projects\n", cyan("fix"))
	fmt.Printf("    %s               Show version\n", cyan("version"))
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Printf("    %s rxgo new my-app\n", gray("$"))
	fmt.Printf("    %s rxgo serve\n", gray("$"))
	fmt.Printf("    %s rxgo generate component ProductCard\n", gray("$"))
	fmt.Printf("    %s rxgo generate page About\n", gray("$"))
	fmt.Printf("    %s rxgo build\n", gray("$"))
	fmt.Printf("    %s rxgo mcp serve\n\n", gray("$"))
}

func step(action, path string) {
	fmt.Printf("  %s %s %s\n", green("✓"), gray(action), path)
}

func fatal(msg string) {
	fmt.Printf("\n  %s %s\n\n", red("✗"), msg)
	os.Exit(1)
}

func renderTemplate(tmpl string, data map[string]string) string {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return tmpl
	}
	var sb strings.Builder
	t.Execute(&sb, data)
	return sb.String()
}

// ANSI colors
func green(s string) string  { return "\033[32m" + s + "\033[0m" }
func cyan(s string) string   { return "\033[36m" + s + "\033[0m" }
func gray(s string) string   { return "\033[90m" + s + "\033[0m" }
func red(s string) string    { return "\033[31m" + s + "\033[0m" }

// ---------------------------------------------------------
// Embedded templates
// ---------------------------------------------------------

const mainGoTemplate = `package main

import (
	rx "github.com/ahmad-nexarapp/ryxogo"
	"{{.ModuleName}}/pages"
)

func main() {
	app := rx.New()
	app.Route("/", &pages.HomePage{})
	app.Run()
}
`

const pagesIndexTemplate = `package pages

import (
	"strconv"
	rx "github.com/ahmad-nexarapp/ryxogo"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

type HomePage struct {
	rx.Page
	count *signal.Signal[int]
}

func (p *HomePage) Setup() {
	p.count = rx.Use(0)
}

func (p *HomePage) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "min-h-screen bg-gray-50 flex items-center justify-center"},
		rx.Div(rx.Props{Class: "bg-white rounded-2xl shadow-sm border p-10 text-center max-w-md w-full"},
			rx.H1(rx.Props{Class: "text-3xl font-bold text-gray-900 mb-6"},
				rx.Text("Welcome to RyxoGo ⚡"),
			),
			rx.P(rx.Props{Class: "text-4xl font-bold text-indigo-600 mb-6"},
				rx.Text(strconv.Itoa(p.count.Val())),
			),
			rx.Button(rx.Props{
				Class:   "px-6 py-2 bg-indigo-600 text-white rounded-lg font-medium",
				OnClick: func() { p.count.Set(p.count.Val() + 1) },
			}, rx.Text("Click me")),
		),
	)
}
`

const goModTemplate = `module {{.ModuleName}}

go 1.22

require github.com/ahmad-nexarapp/ryxogo v0.1.2
`

const gitignoreTemplate = `dist/
*.wasm
.DS_Store
`

const readmeTemplate = `# {{.AppName}}

Built with RyxoGo.

## Start

` + "```bash\nrxgo serve\n```"

const cursorRulesTemplate = `# RyxoGo Rules — {{.AppName}}
# This project uses RyxoGo (Go → WASM), NOT React/Vue/JS.
# MCP server: rxgo mcp serve → http://localhost:7777
# Always call get_app_schema before generating code.
# Signals: rx.Use(), rx.Computed(), rx.Async() — no useState
# Elements: rx.Div(), rx.Button(), rx.Text() — no JSX
`

const claudeMDTemplate = `# {{.AppName}} — RyxoGo Project
## MCP: rxgo mcp serve → http://localhost:7777
## Framework: RyxoGo (Go → WebAssembly)
## State: rx.Use(v), rx.Computed(fn), rx.Async(fn)
## Elements: rx.Div/Button/Input/Text etc
## Routing: file-based in pages/, register in main.go
`

const agentsMDTemplate = `# AGENTS.md — {{.AppName}}
# Framework: RyxoGo (Go → WASM). NOT JavaScript.
# MCP: GET http://localhost:7777/schema for app structure
# Pattern: struct + Setup() + Render() — always this structure
# State: rx.Use() signals only, never raw variables
`

const defaultIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>RyxoGo App</title>
  <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
  <link rel="stylesheet" href="/styles.css" />
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,-apple-system,sans-serif;background:#f9fafb;color:#111827}
    #ryxogo-loading{display:flex;align-items:center;justify-content:center;height:100vh;color:#6b7280;gap:10px;font-size:14px}
    .sp{width:18px;height:18px;border:2px solid #e5e7eb;border-top-color:#4f46e5;border-radius:50%;animation:spin .6s linear infinite}
    @keyframes spin{to{transform:rotate(360deg)}}
  </style>
</head>
<body>
  <div id="app">
    <div id="ryxogo-loading"><div class="sp"></div><span>Loading...</span></div>
  </div>
  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('/app.wasm?v=' + Date.now()), go.importObject)
      .then(r => { document.getElementById('ryxogo-loading')?.remove(); go.run(r.instance); })
      .catch(e => { document.getElementById('app').innerHTML = '<div style="padding:2rem;color:#dc2626;font-family:monospace">RyxoGo load error:<br>'+e+'</div>'; });
  </script>
</body>
</html>`

const defaultCSS = `/* RyxoGo default styles — replaces Tailwind CDN */
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f9fafb;color:#111827;line-height:1.5}
.min-h-screen{min-height:100vh}
.flex{display:flex}.items-center{align-items:center}.justify-center{justify-content:center}
.flex-col{flex-direction:column}.gap-3{gap:.75rem}.gap-4{gap:1rem}
.p-4{padding:1rem}.p-6{padding:1.5rem}.p-8{padding:2rem}.p-10{padding:2.5rem}
.px-3{padding-left:.75rem;padding-right:.75rem}.px-4{padding-left:1rem;padding-right:1rem}
.px-6{padding-left:1.5rem;padding-right:1.5rem}.py-2{padding-top:.5rem;padding-bottom:.5rem}
.mb-2{margin-bottom:.5rem}.mb-4{margin-bottom:1rem}.mb-6{margin-bottom:1.5rem}.mb-8{margin-bottom:2rem}
.mt-4{margin-top:1rem}.ml-4{margin-left:1rem}.mx-auto{margin-left:auto;margin-right:auto}
.w-full{width:100%}.max-w-md{max-width:28rem}.max-w-lg{max-width:32rem}
.text-sm{font-size:.875rem}.text-lg{font-size:1.125rem}.text-xl{font-size:1.25rem}
.text-2xl{font-size:1.5rem}.text-3xl{font-size:1.875rem}.text-4xl{font-size:2.25rem}
.font-medium{font-weight:500}.font-semibold{font-weight:600}.font-bold{font-weight:700}
.text-center{text-align:center}.text-left{text-align:left}
.text-white{color:#fff}.text-gray-500{color:#6b7280}.text-gray-700{color:#374151}
.text-gray-900{color:#111827}.text-indigo-600{color:#4f46e5}.text-red-600{color:#dc2626}
.text-red-700{color:#b91c1c}.text-green-600{color:#16a34a}
.bg-white{background:#fff}.bg-gray-50{background:#f9fafb}.bg-gray-100{background:#f3f4f6}
.bg-gray-200{background:#e5e7eb}.bg-indigo-600{background:#4f46e5}.bg-indigo-700{background:#4338ca}
.bg-red-100{background:#fee2e2}.bg-blue-600{background:#2563eb}
.border{border:1px solid #e5e7eb}.border-t{border-top:1px solid #e5e7eb}
.rounded{border-radius:.25rem}.rounded-lg{border-radius:.5rem}.rounded-xl{border-radius:.75rem}
.rounded-2xl{border-radius:1rem}.rounded-full{border-radius:9999px}
.shadow-sm{box-shadow:0 1px 2px rgba(0,0,0,.05)}
.grid{display:grid}.gap-3{gap:.75rem}
.cursor-pointer{cursor:pointer}
.opacity-50{opacity:.5}.disabled\:opacity-50:disabled{opacity:.5}
button{cursor:pointer;border:none;font-family:inherit;font-size:inherit;transition:opacity .15s}
button:hover{opacity:.9}
input,textarea,select{font-family:inherit;font-size:inherit;border:1px solid #d1d5db;border-radius:.375rem;padding:.5rem .75rem;width:100%;outline:none}
input:focus,textarea:focus{border-color:#4f46e5;box-shadow:0 0 0 2px rgba(79,70,229,.2)}
`

const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect width="32" height="32" rx="6" fill="#4f46e5"/>
  <text x="16" y="22" text-anchor="middle" font-size="18" fill="#fff" font-family="system-ui">⚡</text>
</svg>`
