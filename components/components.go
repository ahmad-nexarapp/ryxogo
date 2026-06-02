// Package components provides a ready-made UI component library for RyxoGo.
// Import and use directly — no need to build common UI from scratch.
//
//	import ui "github.com/ahmad-nexarapp/ryxogo/components"
//
//	ui.Button(ui.ButtonProps{Label: "Save", Variant: "primary", OnClick: save})
//	ui.Card(ui.CardProps{Title: "Stats"}, content...)
package components

import (
	"github.com/ahmad-nexarapp/ryxogo/core"
)

type Node = core.Node
type Props = core.Props

// ---------------------------------------------------------
// Button
// ---------------------------------------------------------

type ButtonProps struct {
	Label    string
	OnClick  func()
	Variant  string // "primary" | "secondary" | "danger" | "ghost"
	Size     string // "sm" | "md" | "lg"
	Disabled bool
	FullWidth bool
}

func Button(p ButtonProps) *Node {
	variant := p.Variant
	if variant == "" {
		variant = "primary"
	}
	size := p.Size
	if size == "" {
		size = "md"
	}

	class := "rxui-btn rxui-btn-" + variant + " rxui-btn-" + size
	if p.FullWidth {
		class += " rxui-btn-full"
	}

	return core.Button(Props{
		Class:    class,
		OnClick:  p.OnClick,
		Disabled: p.Disabled,
	}, core.Text(p.Label))
}

// ---------------------------------------------------------
// Card
// ---------------------------------------------------------

type CardProps struct {
	Title    string
	Subtitle string
	Class    string
}

func Card(p CardProps, children ...*Node) *Node {
	class := "rxui-card"
	if p.Class != "" {
		class += " " + p.Class
	}

	contents := []*Node{}
	if p.Title != "" {
		contents = append(contents, core.Div(Props{Class: "rxui-card-header"},
			core.H3(Props{Class: "rxui-card-title"}, core.Text(p.Title)),
			core.IfOnly(p.Subtitle != "",
				core.P(Props{Class: "rxui-card-subtitle"}, core.Text(p.Subtitle)),
			),
		))
	}
	contents = append(contents, core.Div(Props{Class: "rxui-card-body"}, children...))

	return core.Div(Props{Class: class}, contents...)
}

// ---------------------------------------------------------
// Input
// ---------------------------------------------------------

type InputProps struct {
	Value       string
	Placeholder string
	Type        string
	Label       string
	Error       string
	OnInput     func(string)
	Disabled    bool
}

func Input(p InputProps) *Node {
	inputType := p.Type
	if inputType == "" {
		inputType = "text"
	}

	class := "rxui-input"
	if p.Error != "" {
		class += " rxui-input-error"
	}

	field := []*Node{}
	if p.Label != "" {
		field = append(field, core.Label(Props{Class: "rxui-label"}, core.Text(p.Label)))
	}
	field = append(field, core.Input(Props{
		Class:       class,
		Value:       p.Value,
		Placeholder: p.Placeholder,
		Type:        inputType,
		OnInput:     p.OnInput,
		Disabled:    p.Disabled,
	}))
	if p.Error != "" {
		field = append(field, core.Span(Props{Class: "rxui-error-text"}, core.Text(p.Error)))
	}

	return core.Div(Props{Class: "rxui-field"}, field...)
}

// ---------------------------------------------------------
// Badge
// ---------------------------------------------------------

type BadgeProps struct {
	Label   string
	Variant string // "success" | "warning" | "danger" | "info" | "neutral"
}

func Badge(p BadgeProps) *Node {
	variant := p.Variant
	if variant == "" {
		variant = "neutral"
	}
	return core.Span(Props{Class: "rxui-badge rxui-badge-" + variant},
		core.Text(p.Label))
}

// ---------------------------------------------------------
// Alert
// ---------------------------------------------------------

type AlertProps struct {
	Message string
	Variant string // "success" | "warning" | "danger" | "info"
	Title   string
}

func Alert(p AlertProps) *Node {
	variant := p.Variant
	if variant == "" {
		variant = "info"
	}
	contents := []*Node{}
	if p.Title != "" {
		contents = append(contents, core.Strong(Props{Class: "rxui-alert-title"}, core.Text(p.Title)))
	}
	contents = append(contents, core.Span(Props{}, core.Text(p.Message)))
	return core.Div(Props{Class: "rxui-alert rxui-alert-" + variant}, contents...)
}

// ---------------------------------------------------------
// Spinner
// ---------------------------------------------------------

func Spinner() *Node {
	return core.Div(Props{Class: "rxui-spinner"})
}

// ---------------------------------------------------------
// Modal
// ---------------------------------------------------------

type ModalProps struct {
	Title   string
	Open    bool
	OnClose func()
}

func Modal(p ModalProps, children ...*Node) *Node {
	if !p.Open {
		return core.Fragment()
	}

	body := []*Node{
		core.Div(Props{Class: "rxui-modal-header"},
			core.H3(Props{Class: "rxui-modal-title"}, core.Text(p.Title)),
			core.Button(Props{
				Class:   "rxui-modal-close",
				OnClick: p.OnClose,
			}, core.Text("×")),
		),
		core.Div(Props{Class: "rxui-modal-body"}, children...),
	}

	return core.Div(Props{Class: "rxui-modal-overlay", OnClick: p.OnClose},
		core.Div(Props{Class: "rxui-modal"}, body...),
	)
}

// ---------------------------------------------------------
// Stack & Row — layout helpers
// ---------------------------------------------------------

func Stack(gap string, children ...*Node) *Node {
	return core.Div(Props{
		Class: "rxui-stack",
		Style: map[string]string{"gap": gap},
	}, children...)
}

func Row(gap string, children ...*Node) *Node {
	return core.Div(Props{
		Class: "rxui-row",
		Style: map[string]string{"gap": gap},
	}, children...)
}
