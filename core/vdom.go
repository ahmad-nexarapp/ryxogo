// Package core contains the Virtual DOM and rendering engine for RyxoGo.
// This is what turns Go structs into real browser DOM elements.
package core

// ---------------------------------------------------------
// Node — the basic unit of RyxoGo UI
// ---------------------------------------------------------

// NodeType identifies what kind of node this is
type NodeType int

const (
	ElementNode  NodeType = iota // <div>, <p>, <button> etc
	TextNode                     // plain text content
	ComponentNode                // a RyxoGo component
	FragmentNode                 // multiple root elements
)

// Node is a virtual DOM node — a lightweight description of UI.
// RyxoGo compares old and new Node trees to figure out
// the minimal set of real DOM changes needed.
type Node struct {
	Type       NodeType
	Tag        string            // "div", "p", "button", "input" etc
	Text       string            // for TextNode
	Props      Props             // all attributes and event handlers
	Children   []*Node           // child nodes
	Key        string            // for efficient list diffing
	Component  Component         // for ComponentNode
	DOMRef     interface{}       // reference to real DOM element (js.Value in WASM)
	reactive   *ReactiveText     // fine-grained binding; nil for static text nodes
}

// Reactive returns the node's fine-grained binding, or nil if static.
// Used by the renderer to set up per-node effects.
func (n *Node) Reactive() *ReactiveText { return n.reactive }

// Props holds all attributes, classes, styles, and event handlers for a node
type Props struct {
	// Standard HTML attributes
	ID          string
	Class       string
	Style       map[string]string
	Attrs       map[string]string

	// Event handlers
	OnClick     func()
	OnInput     func(value string)
	OnChange    func(value string)
	OnSubmit    func()
	OnFocus     func()
	OnBlur      func()
	OnKeyDown   func(key string)
	OnKeyUp     func(key string)
	OnMouseOver func()
	OnMouseOut  func()

	// Form
	Value       string
	Placeholder string
	Disabled    bool
	Checked     bool
	Type        string // input type
	Src         string
	Alt         string
	Href        string
	Target      string

	// Extra arbitrary attributes
	Data        map[string]string // data-* attributes

	// Key for efficient list diffing — set this when rendering lists
	Key         string
}

// ---------------------------------------------------------
// Builder functions — the simple API developers use
// ---------------------------------------------------------

// El creates an element node with the given tag, props, and children
func El(tag string, props Props, children ...*Node) *Node {
	return &Node{
		Type:     ElementNode,
		Tag:      tag,
		Props:    props,
		Children: children,
	}
}

// Text creates a text node
func Text(content string) *Node {
	return &Node{
		Type: TextNode,
		Text: content,
	}
}

// Fragment wraps multiple nodes without a container element
func Fragment(children ...*Node) *Node {
	return &Node{
		Type:     FragmentNode,
		Children: children,
	}
}

// ---------------------------------------------------------
// Common element helpers — what developers actually type
// ---------------------------------------------------------

func Div(props Props, children ...*Node) *Node {
	return El("div", props, children...)
}

func Span(props Props, children ...*Node) *Node {
	return El("span", props, children...)
}

func P(props Props, children ...*Node) *Node {
	return El("p", props, children...)
}

func H1(props Props, children ...*Node) *Node {
	return El("h1", props, children...)
}

func H2(props Props, children ...*Node) *Node {
	return El("h2", props, children...)
}

func H3(props Props, children ...*Node) *Node {
	return El("h3", props, children...)
}

func Button(props Props, children ...*Node) *Node {
	return El("button", props, children...)
}

func Input(props Props) *Node {
	return El("input", props)
}

func Form(props Props, children ...*Node) *Node {
	return El("form", props, children...)
}

func Img(props Props) *Node {
	return El("img", props)
}

func A(props Props, children ...*Node) *Node {
	return El("a", props, children...)
}

func Ul(props Props, children ...*Node) *Node {
	return El("ul", props, children...)
}

func Li(props Props, children ...*Node) *Node {
	return El("li", props, children...)
}

func Nav(props Props, children ...*Node) *Node {
	return El("nav", props, children...)
}

func Header(props Props, children ...*Node) *Node {
	return El("header", props, children...)
}

func Main(props Props, children ...*Node) *Node {
	return El("main", props, children...)
}

func Footer(props Props, children ...*Node) *Node {
	return El("footer", props, children...)
}

func Section(props Props, children ...*Node) *Node {
	return El("section", props, children...)
}

func Article(props Props, children ...*Node) *Node {
	return El("article", props, children...)
}

// Additional elements
func Label(props Props, children ...*Node) *Node {
	return El("label", props, children...)
}

func Textarea(props Props, children ...*Node) *Node {
	return El("textarea", props, children...)
}

func Select(props Props, children ...*Node) *Node {
	return El("select", props, children...)
}

func Option(props Props, children ...*Node) *Node {
	return El("option", props, children...)
}

func Table(props Props, children ...*Node) *Node {
	return El("table", props, children...)
}

func Thead(props Props, children ...*Node) *Node {
	return El("thead", props, children...)
}

func Tbody(props Props, children ...*Node) *Node {
	return El("tbody", props, children...)
}

func Tr(props Props, children ...*Node) *Node {
	return El("tr", props, children...)
}

func Th(props Props, children ...*Node) *Node {
	return El("th", props, children...)
}

func Td(props Props, children ...*Node) *Node {
	return El("td", props, children...)
}

func Pre(props Props, children ...*Node) *Node {
	return El("pre", props, children...)
}

func Code(props Props, children ...*Node) *Node {
	return El("code", props, children...)
}

func Hr(props Props) *Node {
	return El("hr", props)
}

func Br(props Props) *Node {
	return El("br", props)
}

func Strong(props Props, children ...*Node) *Node {
	return El("strong", props, children...)
}

func Em(props Props, children ...*Node) *Node {
	return El("em", props, children...)
}

func Small(props Props, children ...*Node) *Node {
	return El("small", props, children...)
}

func Ol(props Props, children ...*Node) *Node {
	return El("ol", props, children...)
}

func Aside(props Props, children ...*Node) *Node {
	return El("aside", props, children...)
}

// ---------------------------------------------------------
// Helpers — conditional rendering, list rendering
// ---------------------------------------------------------

// If renders one of two nodes based on a condition.
// Replaces long if/else blocks in Render().
// if(isLoggedIn, Dashboard{}, LoginPage{})
func If(condition bool, yes *Node, no *Node) *Node {
	if condition {
		return yes
	}
	return no
}

// IfOnly renders a node only when condition is true, otherwise nothing
func IfOnly(condition bool, node *Node) *Node {
	if condition {
		return node
	}
	return nil
}

// Each renders a list of nodes from a slice.
// each(users, func(u User) *Node { return UserCard{user: u}.Render() })
func Each[T any](items []T, fn func(item T, index int) *Node) []*Node {
	nodes := make([]*Node, 0, len(items))
	for i, item := range items {
		node := fn(item, i)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Nodes flattens a slice of nodes into variadic children
func Nodes(nodes []*Node) []*Node {
	result := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if n != nil {
			result = append(result, n)
		}
	}
	return result
}
