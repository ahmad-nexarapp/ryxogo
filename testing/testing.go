// Package testing provides test utilities for RyxoGo components.
// Render components to HTML and assert on the output — no browser needed.
//
//	import rxtest "github.com/ahmad-nexarapp/ryxogo/testing"
//
//	func TestButton(t *testing.T) {
//	    r := rxtest.Render(&MyComponent{})
//	    r.AssertText(t, "Click me")
//	    r.AssertHasClass(t, "btn-primary")
//	}
package testing

import (
	"strings"
	"testing"

	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/ssr"
)

// Result wraps rendered HTML for assertions.
type Result struct {
	HTML string
	node *core.Node
}

// Render renders a component to a testable Result.
// Calls Setup() automatically.
func Render(comp core.Component) *Result {
	if s, ok := comp.(interface{ Setup() }); ok {
		s.Setup()
	}
	node := comp.Render()
	return &Result{
		HTML: ssr.RenderNode(node),
		node: node,
	}
}

// RenderNoSetup renders without calling Setup() — use when you've already
// called Setup() yourself and want to test state after interactions.
//
//	c := &Counter{}
//	c.Setup()
//	r := rxtest.Render(c)        // first render
//	r.Click("inc")               // simulate click
//	r2 := rxtest.RenderNoSetup(c) // re-render WITHOUT resetting state
//	r2.AssertText(t, ">1<")
func RenderNoSetup(comp core.Component) *Result {
	node := comp.Render()
	return &Result{
		HTML: ssr.RenderNode(node),
		node: node,
	}
}

// AssertText fails the test if the HTML doesn't contain the given text.
func (r *Result) AssertText(t *testing.T, text string) {
	t.Helper()
	if !strings.Contains(r.HTML, text) {
		t.Errorf("expected HTML to contain text %q\nGot: %s", text, r.HTML)
	}
}

// AssertNoText fails if the HTML contains the given text.
func (r *Result) AssertNoText(t *testing.T, text string) {
	t.Helper()
	if strings.Contains(r.HTML, text) {
		t.Errorf("expected HTML NOT to contain %q\nGot: %s", text, r.HTML)
	}
}

// AssertHasClass fails if no element has the given class.
func (r *Result) AssertHasClass(t *testing.T, class string) {
	t.Helper()
	if !strings.Contains(r.HTML, class) {
		t.Errorf("expected an element with class %q\nGot: %s", class, r.HTML)
	}
}

// AssertHasTag fails if the HTML doesn't contain the given tag.
func (r *Result) AssertHasTag(t *testing.T, tag string) {
	t.Helper()
	open := "<" + tag
	if !strings.Contains(r.HTML, open) {
		t.Errorf("expected a <%s> element\nGot: %s", tag, r.HTML)
	}
}

// AssertAttr fails if the HTML doesn't contain attr="value".
func (r *Result) AssertAttr(t *testing.T, attr, value string) {
	t.Helper()
	expected := attr + `="` + value + `"`
	if !strings.Contains(r.HTML, expected) {
		t.Errorf("expected attribute %s\nGot: %s", expected, r.HTML)
	}
}

// CountTag returns how many times a tag appears.
func (r *Result) CountTag(tag string) int {
	return strings.Count(r.HTML, "<"+tag)
}

// AssertTagCount fails if the tag count doesn't match.
func (r *Result) AssertTagCount(t *testing.T, tag string, want int) {
	t.Helper()
	got := r.CountTag(tag)
	if got != want {
		t.Errorf("expected %d <%s> elements, got %d\nHTML: %s", want, tag, got, r.HTML)
	}
}

// FindClicker walks the virtual DOM and returns the OnClick handler of the
// first element matching the given class. Lets you simulate clicks in tests.
func (r *Result) FindClicker(class string) func() {
	return findClick(r.node, class)
}

func findClick(node *core.Node, class string) func() {
	if node == nil {
		return nil
	}
	if node.Type == core.ElementNode {
		if strings.Contains(node.Props.Class, class) && node.Props.OnClick != nil {
			return node.Props.OnClick
		}
	}
	for _, child := range node.Children {
		if fn := findClick(child, class); fn != nil {
			return fn
		}
	}
	return nil
}

// Click simulates clicking the first element with the given class.
// Returns true if an element was found and clicked.
func (r *Result) Click(class string) bool {
	if fn := r.FindClicker(class); fn != nil {
		fn()
		return true
	}
	return false
}
