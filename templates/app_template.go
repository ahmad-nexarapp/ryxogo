package templates

// AppTemplate holds all the files that rxgo new generates
var AppTemplate = map[string]string{

	// -------------------------------------------------------
	// main.go — app entry point
	// -------------------------------------------------------
	"main.go": `package main

import (
	rx "github.com/ahmad-nexarapp/ryxogo"
	"{{.ModuleName}}/pages"
)

func main() {
	app := rx.New()

	// Register your pages here
	// File-based routing: pages/about.go → /about
	app.Route("/", &pages.HomePage{})

	app.Run()
}
`,

	// -------------------------------------------------------
	// pages/index.go — home page
	// -------------------------------------------------------
	"pages/index.go": `package pages

import (
	"strconv"
	rx "github.com/ahmad-nexarapp/ryxogo"
)

// HomePage is the main landing page at route "/"
type HomePage struct {
	rx.Page
	count *rx.Signal[int]
}

func (p *HomePage) Setup() {
	p.count = rx.Use(0)
}

func (p *HomePage) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "min-h-screen bg-gray-50 flex items-center justify-center"},
		rx.Div(rx.Props{Class: "bg-white rounded-2xl shadow-sm border p-10 text-center max-w-md w-full"},

			rx.Div(rx.Props{Class: "text-5xl mb-4"}, rx.Text("⚡")),

			rx.H1(rx.Props{Class: "text-3xl font-bold text-gray-900 mb-2"},
				rx.Text("Welcome to RyxoGo"),
			),

			rx.P(rx.Props{Class: "text-gray-500 mb-8"},
				rx.Text("Your Go-first frontend framework"),
			),

			rx.P(rx.Props{Class: "text-4xl font-bold text-indigo-600 mb-6"},
				rx.Text(strconv.Itoa(p.count.Val())),
			),

			rx.Div(rx.Props{Class: "flex gap-3 justify-center"},
				rx.Button(rx.Props{
					Class:   "px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium",
					OnClick: func() { p.count.Set(p.count.Val() + 1) },
				}, rx.Text("Click me")),

				rx.Button(rx.Props{
					Class:   "px-6 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 font-medium",
					OnClick: func() { p.count.Set(0) },
				}, rx.Text("Reset")),
			),
		),
	)
}
`,

	// -------------------------------------------------------
	// go.mod
	// -------------------------------------------------------
	"go.mod": `module {{.ModuleName}}

go 1.22

require github.com/ahmad-nexarapp/ryxogo v0.1.0
`,

	// -------------------------------------------------------
	// .gitignore
	// -------------------------------------------------------
	".gitignore": `# RyxoGo build output
dist/
*.wasm

# Go
*.exe
*.test

# OS
.DS_Store
Thumbs.db
`,

	// -------------------------------------------------------
	// README.md
	// -------------------------------------------------------
	"README.md": `# {{.AppName}}

Built with [RyxoGo](https://github.com/ahmad-nexarapp/ryxogo) — a Go-first frontend framework.

## Getting started

` + "```" + `bash
rxgo serve        # dev server at localhost:3000
rxgo build        # build for production → dist/
rxgo mcp serve    # start AI tools MCP server
` + "```" + `

## Project structure

` + "```" + `
{{.AppName}}/
├── pages/          # route components (file = URL)
├── components/     # reusable UI components
├── stores/         # global state
├── types/          # shared Go types
├── public/         # static files
└── main.go         # app entry point
` + "```" + `

## Adding a page

Create ` + "`pages/about.go`" + ` → becomes route ` + "`/about`" + `

` + "```go" + `
package pages

import rx "github.com/ahmad-nexarapp/ryxogo"

type AboutPage struct{ rx.Page }

func (p *AboutPage) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "p-8"},
		rx.H1(rx.Props{}, rx.Text("About")),
	)
}
` + "```" + `

Then register in ` + "`main.go`" + `:
` + "```go" + `
app.Route("/about", &pages.AboutPage{})
` + "```" + `
`,
}
