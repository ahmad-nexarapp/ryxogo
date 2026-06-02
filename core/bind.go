// bind.go — two-way input binding for RyxoGo.
// Eliminates the getInputValue workaround completely.
package core

// Bindable is implemented by signals that support two-way binding.
type Bindable[T any] interface {
	Val() T
	Set(T)
}

// BindString creates Props for a text input bound to a string signal.
// Replaces the manual Value + OnInput pattern.
//
//	// Instead of:
//	rx.Input(rx.Props{Value: p.name.Val(), OnInput: func(v string) { p.name.Set(v) }})
//
//	// Write:
//	rx.Input(rx.Bind(p.name, rx.Props{Placeholder: "Your name", Class: "border rounded"}))
func BindString[T interface{ Val() string; Set(string) }](sig T, extra Props) Props {
	extra.Value = sig.Val()
	extra.OnInput = func(v string) { sig.Set(v) }
	return extra
}

// Bind creates Props for any input type.
// For string signals use BindString — it's simpler.
// For numeric inputs use BindInt or BindFloat.
func Bind(val string, onInput func(string), extra Props) Props {
	extra.Value = val
	extra.OnInput = onInput
	return extra
}
