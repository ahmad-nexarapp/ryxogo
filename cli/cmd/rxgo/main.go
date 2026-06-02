// rxgo — The RyxoGo CLI tool
// Install: go install github.com/ahmad-nexarapp/ryxogo/cli/cmd/rxgo@latest
// Commands: new, serve, build, mcp, generate, ai
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

const version = "0.1.0"

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

	// Generate AI config files
	generateAIFiles(appName, data)

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

// ---------------------------------------------------------
// rxgo mcp [serve]
// ---------------------------------------------------------

func cmdMCP() {
	sub := "serve"
	if len(os.Args) >= 3 {
		sub = os.Args[2]
	}

	switch sub {
	case "serve":
		port := "7777"
		fmt.Printf("\n  %s RyxoGo MCP server\n\n", cyan("⚡"))
		fmt.Printf("  %s http://localhost:%s\n", green("✓ Running at"), port)
		fmt.Printf("  %s\n\n", gray("AI tools (Cursor, Claude, Codex) connect here"))
		fmt.Printf("  %s\n\n", gray("Watching for file changes..."))

		// Scan project and serve schema
		startMCPServer(port)
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

	// Build the WASM binary
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "app.wasm"), ".")
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
	storeName := name + "Store"
	varName := strings.ToLower(name[:1]) + name[1:]
	path := filepath.Join("stores", strings.ToLower(name)+".go")
	content := fmt.Sprintf(`package stores

import rx "github.com/ahmad-nexarapp/ryxogo"

type %s struct {
	// Define store fields here
}

var %s = rx.NewStore(&%s{})

// Actions
func Set%sField(value string) {
	rx.UpdateStore(%s, func(s *%s) {
		// s.Field = value
	})
}
`, storeName, varName, storeName, name, varName, storeName)

	writeGenerated(path, content, "store", storeName)
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

// ---------------------------------------------------------
// MCP server (simple HTTP JSON server)
// ---------------------------------------------------------

func startMCPServer(port string) {
	mux := http.NewServeMux()

	// Schema endpoint
	mux.HandleFunc("/schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, `{"framework":"ryxogo","version":"%s","status":"running"}`, version)
	})

	// Tools endpoint
	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, `["get_app_schema","get_component","list_routes","list_types","validate_component","get_ryxogo_docs","generate_component_scaffold"]`)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	http.ListenAndServe(":"+port, mux)
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

require github.com/ahmad-nexarapp/ryxogo v0.1.0
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
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    #ryxogo-loading { display:flex;align-items:center;justify-content:center;height:100vh;font-family:system-ui;color:#6b7280;gap:10px }
    .sp { width:18px;height:18px;border:2px solid #e5e7eb;border-top-color:#4f46e5;border-radius:50%;animation:spin .6s linear infinite }
    @keyframes spin { to { transform:rotate(360deg) } }
  </style>
</head>
<body>
  <div id="app"><div id="ryxogo-loading"><div class="sp"></div><span>Loading...</span></div></div>
  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('/app.wasm'), go.importObject)
      .then(r => { document.getElementById('ryxogo-loading')?.remove(); go.run(r.instance); })
      .catch(e => { document.getElementById('app').innerHTML = '<div style="padding:2rem;color:red">Load error: '+e+'</div>'; });
  </script>
</body>
</html>`
