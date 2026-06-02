// ssg.go — static site generation for RyxoGo.
// Renders routes to .html files at build time for deployment to any CDN.
// Best of both worlds: static HTML for SEO + WASM hydration for interactivity.
package ssr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// StaticPage describes a page to pre-render.
type StaticPage struct {
	Path      string // URL path e.g. "/about"
	Component core.Component
	Meta      Meta
}

// Generator produces static HTML files.
type Generator struct {
	server *Server
	outDir string
	pages  []StaticPage
}

// NewGenerator creates an SSG generator writing to outDir.
func NewGenerator(outDir string) *Generator {
	return &Generator{
		server: NewServer(),
		outDir: outDir,
	}
}

// AddPage registers a page to pre-render.
func (g *Generator) AddPage(path string, comp core.Component, meta Meta) *Generator {
	g.pages = append(g.pages, StaticPage{Path: path, Component: comp, Meta: meta})
	return g
}

// Generate renders all registered pages to HTML files.
func (g *Generator) Generate() error {
	if err := os.MkdirAll(g.outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	for _, page := range g.pages {
		if err := g.renderPage(page); err != nil {
			return fmt.Errorf("rendering %s: %w", page.Path, err)
		}
	}
	return nil
}

func (g *Generator) renderPage(page StaticPage) error {
	content := RenderToString(page.Component)

	meta := page.Meta
	if meta.URL == "" {
		meta.URL = page.Path
	}

	var sb strings.Builder
	writeShell(&sb, content, meta)

	// Determine output file path
	var outPath string
	if page.Path == "/" {
		outPath = filepath.Join(g.outDir, "index.html")
	} else {
		dir := filepath.Join(g.outDir, strings.Trim(page.Path, "/"))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		outPath = filepath.Join(dir, "index.html")
	}

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

// writeShell writes the full HTML document (shared with server.go logic).
func writeShell(sb *strings.Builder, content string, meta Meta) {
	if meta.Type == "" {
		meta.Type = "website"
	}
	fmt.Fprintf(sb, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
  <meta name="description" content="%s" />
  <meta name="keywords" content="%s" />
  <meta property="og:title" content="%s" />
  <meta property="og:description" content="%s" />
  <meta property="og:type" content="%s" />
  <meta property="og:url" content="%s" />
  <meta property="og:image" content="%s" />
  <meta name="twitter:card" content="summary_large_image" />
  <link rel="stylesheet" href="/styles.css" />
  <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
</head>
<body>
  <div id="app" data-ssr="true">%s</div>
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
		content)
}
