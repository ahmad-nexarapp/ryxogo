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
		// Reactive text nodes: compute the current value once for the
		// server-rendered HTML. The client effect takes over on hydration.
		if rt := node.Reactive(); rt != nil && rt.Compute != nil {
			sb.WriteString(html.EscapeString(rt.Compute()))
		} else {
			sb.WriteString(html.EscapeString(node.Text))
		}
	case core.FragmentNode:
		for _, child := range node.Children {
			renderNode(sb, child)
		}
	case core.ElementNode:
		sb.WriteString("<")
		sb.WriteString(node.Tag)
		renderAttrs(sb, node.Props)
		// Fine-grained bindings: render their initial computed values into
		// the server HTML so the page is correct before hydration.
		if node.HasBindings() {
			renderBindings(sb, node.BindingSet())
		}
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

// renderBindings writes the initial computed values of fine-grained
// attribute/style/visibility bindings into the server-rendered HTML.
func renderBindings(sb *strings.Builder, b *core.BindingSetView) {
	styleParts := map[string]string{}
	for _, attr := range b.Attrs() {
		if attr.Compute == nil {
			continue
		}
		val := attr.Compute()
		switch {
		case attr.Name == "class":
			fmt.Fprintf(sb, ` class="%s"`, html.EscapeString(val))
		case attr.Name == "disabled":
			if val == "true" {
				sb.WriteString(" disabled")
			}
		case attr.Name == "checked":
			if val == "true" {
				sb.WriteString(" checked")
			}
		case len(attr.Name) > 6 && attr.Name[:6] == "style:":
			styleParts[attr.Name[6:]] = val
		default:
			fmt.Fprintf(sb, ` %s="%s"`, attr.Name, html.EscapeString(val))
		}
	}
	// Visibility: render display:none into the initial style if hidden
	hidden := false
	if show := b.Show(); show != nil && show.Compute != nil {
		hidden = !show.Compute()
	}
	if len(styleParts) > 0 || hidden {
		sb.WriteString(` style="`)
		first := true
		for prop, val := range styleParts {
			if !first {
				sb.WriteString("; ")
			}
			fmt.Fprintf(sb, "%s: %s", prop, html.EscapeString(val))
			first = false
		}
		if hidden {
			if !first {
				sb.WriteString("; ")
			}
			sb.WriteString("display: none")
		}
		sb.WriteString(`"`)
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
	if p.Name != "" {
		fmt.Fprintf(sb, ` name="%s"`, html.EscapeString(p.Name))
	}
	if p.For != "" {
		fmt.Fprintf(sb, ` for="%s"`, html.EscapeString(p.For))
	}
	if p.AutoComplete != "" {
		fmt.Fprintf(sb, ` autocomplete="%s"`, html.EscapeString(p.AutoComplete))
	}
	if p.Placeholder != "" {
		fmt.Fprintf(sb, ` placeholder="%s"`, html.EscapeString(p.Placeholder))
	}
	if p.Type != "" {
		fmt.Fprintf(sb, ` type="%s"`, html.EscapeString(p.Type))
	}
	if p.Min != "" {
		fmt.Fprintf(sb, ` min="%s"`, html.EscapeString(p.Min))
	}
	if p.Max != "" {
		fmt.Fprintf(sb, ` max="%s"`, html.EscapeString(p.Max))
	}
	if p.Step != "" {
		fmt.Fprintf(sb, ` step="%s"`, html.EscapeString(p.Step))
	}
	if p.Pattern != "" {
		fmt.Fprintf(sb, ` pattern="%s"`, html.EscapeString(p.Pattern))
	}
	if p.Rows != "" {
		fmt.Fprintf(sb, ` rows="%s"`, html.EscapeString(p.Rows))
	}
	if p.Cols != "" {
		fmt.Fprintf(sb, ` cols="%s"`, html.EscapeString(p.Cols))
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
	if p.Required {
		sb.WriteString(` required`)
	}
	if p.ReadOnly {
		sb.WriteString(` readonly`)
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
