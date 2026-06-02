// Package ssr renders RyxoGo components to HTML strings on the server.
// Enables server-side rendering (SSR) and static site generation (SSG)
// for fast first paint and SEO — no browser or WASM needed.
//
// The same component code that runs in the browser via WASM also runs here
// in pure Go, producing HTML sent to the client immediately. The WASM bundle
// then hydrates the static HTML, attaching event handlers.
package ssr

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

// RenderToString renders a component to an HTML string.
func RenderToString(comp core.Component) string {
	if s, ok := comp.(interface{ Setup() }); ok {
		s.Setup()
	}
	node := comp.Render()
	var sb strings.Builder
	renderNode(&sb, node)
	return sb.String()
}

// RenderNode renders a single virtual DOM node to HTML.
func RenderNode(node *core.Node) string {
	var sb strings.Builder
	renderNode(&sb, node)
	return sb.String()
}

func renderNode(sb *strings.Builder, node *core.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case core.TextNode:
		sb.WriteString(html.EscapeString(node.Text))
	case core.FragmentNode:
		for _, child := range node.Children {
			renderNode(sb, child)
		}
	case core.ElementNode:
		sb.WriteString("<")
		sb.WriteString(node.Tag)
		renderAttrs(sb, node.Props)
		if voidElements[node.Tag] {
			sb.WriteString("/>")
			return
		}
		sb.WriteString(">")
		for _, child := range node.Children {
			renderNode(sb, child)
		}
		sb.WriteString("</")
		sb.WriteString(node.Tag)
		sb.WriteString(">")
	}
}

func renderAttrs(sb *strings.Builder, p core.Props) {
	if p.ID != "" {
		fmt.Fprintf(sb, ` id="%s"`, html.EscapeString(p.ID))
	}
	if p.Class != "" {
		fmt.Fprintf(sb, ` class="%s"`, html.EscapeString(p.Class))
	}
	if p.Value != "" {
		fmt.Fprintf(sb, ` value="%s"`, html.EscapeString(p.Value))
	}
	if p.Placeholder != "" {
		fmt.Fprintf(sb, ` placeholder="%s"`, html.EscapeString(p.Placeholder))
	}
	if p.Type != "" {
		fmt.Fprintf(sb, ` type="%s"`, html.EscapeString(p.Type))
	}
	if p.Src != "" {
		fmt.Fprintf(sb, ` src="%s"`, html.EscapeString(p.Src))
	}
	if p.Alt != "" {
		fmt.Fprintf(sb, ` alt="%s"`, html.EscapeString(p.Alt))
	}
	if p.Href != "" {
		fmt.Fprintf(sb, ` href="%s"`, html.EscapeString(p.Href))
	}
	if p.Target != "" {
		fmt.Fprintf(sb, ` target="%s"`, html.EscapeString(p.Target))
	}
	if p.Disabled {
		sb.WriteString(` disabled`)
	}
	if p.Checked {
		sb.WriteString(` checked`)
	}
	if len(p.Style) > 0 {
		keys := sortedKeys(p.Style)
		sb.WriteString(` style="`)
		for i, k := range keys {
			if i > 0 {
				sb.WriteString("; ")
			}
			fmt.Fprintf(sb, "%s: %s", k, html.EscapeString(p.Style[k]))
		}
		sb.WriteString(`"`)
	}
	if len(p.Data) > 0 {
		for _, k := range sortedKeys(p.Data) {
			fmt.Fprintf(sb, ` data-%s="%s"`, k, html.EscapeString(p.Data[k]))
		}
	}
	if len(p.Attrs) > 0 {
		for _, k := range sortedKeys(p.Attrs) {
			fmt.Fprintf(sb, ` %s="%s"`, k, html.EscapeString(p.Attrs[k]))
		}
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
