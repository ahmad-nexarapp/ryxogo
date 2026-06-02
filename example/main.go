// Example RyxoGo app — shows the complete developer experience.
// This is what someone writes when building with RyxoGo.
// Run with: GOARCH=wasm GOOS=js go build -o app.wasm
package main

import (
	"strconv"

	rx "github.com/ahmad-nexarapp/ryxogo"
)

// ---------------------------------------------------------
// Types — shared data types
// ---------------------------------------------------------

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Image string  `json:"image"`
}

// ---------------------------------------------------------
// Counter page — shows signals, computed, events
// ---------------------------------------------------------

type CounterPage struct {
	rx.Page

	// State — declared in Setup(), used in Render()
	count   interface{}  // *signal.Signal[int]
	doubled interface{}  // *signal.Computed[int]
	name    interface{}  // *signal.Signal[string]
}

func (p *CounterPage) Setup() {
	p.count   = rx.Use(0)
	p.name    = rx.Use("world")

	// computed — no dependency array, no bugs
	p.doubled = rx.Computed(func() int {
		return p.count.Val() * 2
	})

	// watch — runs whenever count changes, auto-tracked
	rx.Watch(func() {
		// This runs every time count changes
		// Perfect for: analytics, localStorage, side effects
		_ = p.count.Val() // reading count subscribes us to it
	})
}

func (p *CounterPage) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "p-8 max-w-md mx-auto"},

		rx.H1(rx.Props{Class: "text-2xl font-bold mb-6"},
			rx.Text("Hello, "+p.name.Val()+"!"),
		),

		// Input bound to name signal
		rx.Input(rx.Props{
			Class:       "border rounded px-3 py-2 w-full mb-4",
			Value:       p.name.Val(),
			Placeholder: "Enter your name",
			OnInput: func(v string) {
				p.name.Set(v) // re-renders automatically
			},
		}),

		// Count display
		rx.P(rx.Props{Class: "text-lg mb-2"},
			rx.Text("Count: "+strconv.Itoa(p.count.Val())),
		),

		// Computed value — updates without any extra code
		rx.P(rx.Props{Class: "text-gray-500 mb-4"},
			rx.Text("Doubled: "+strconv.Itoa(p.doubled.Val())),
		),

		// Buttons
		rx.Div(rx.Props{Class: "flex gap-3"},
			rx.Button(rx.Props{
				Class:   "px-4 py-2 bg-blue-600 text-white rounded",
				OnClick: func() { p.count.Set(p.count.Val() + 1) },
			}, rx.Text("+")),

			rx.Button(rx.Props{
				Class:   "px-4 py-2 bg-gray-200 rounded",
				OnClick: func() { p.count.Set(p.count.Val() - 1) },
			}, rx.Text("-")),

			rx.Button(rx.Props{
				Class:   "px-4 py-2 bg-red-100 text-red-700 rounded",
				OnClick: func() { p.count.Set(0) },
			}, rx.Text("Reset")),
		),
	)
}

// ---------------------------------------------------------
// Users page — shows async data fetching
// ---------------------------------------------------------

type UsersPage struct {
	rx.Page
	users *rx.AsyncSignal[[]User]
}

func (p *UsersPage) Setup() {
	// One line. Loading + error + data — all handled.
	p.users = rx.Async(func() ([]User, error) {
		return rx.Get[[]User]("https://jsonplaceholder.typicode.com/users")
	})
}

func (p *UsersPage) Render() *rx.Node {
	// Handle all three states — compiler ensures you handle them
	if p.users.IsLoading() {
		return rx.Div(rx.Props{Class: "p-8"},
			rx.Text("Loading users..."),
		)
	}

	if p.users.IsError() {
		return rx.Div(rx.Props{Class: "p-8 text-red-600"},
			rx.Text("Error: "+p.users.Err().Error()),
			rx.Button(rx.Props{
				Class:   "ml-4 px-3 py-1 border rounded",
				OnClick: p.users.Refetch,
			}, rx.Text("Retry")),
		)
	}

	users := p.users.Data()
	return rx.Div(rx.Props{Class: "p-8"},
		rx.H1(rx.Props{Class: "text-2xl font-bold mb-4"},
			rx.Text("Users ("+strconv.Itoa(len(users))+")"),
		),
		rx.Div(rx.Props{Class: "grid gap-3"},
			rx.Nodes(rx.Each(users, func(u User, _ int) *rx.Node {
				return rx.Div(rx.Props{
					Class: "p-4 border rounded-lg",
					Key:   strconv.Itoa(u.ID), // key for efficient diffing
				},
					rx.P(rx.Props{Class: "font-medium"}, rx.Text(u.Name)),
					rx.P(rx.Props{Class: "text-gray-500 text-sm"}, rx.Text(u.Email)),
				)
			})),
		),
	)
}

// ---------------------------------------------------------
// App entry point
// ---------------------------------------------------------

func main() {
	app := rx.New()

	// File-based routing — or register manually like this
	app.Route("/", &CounterPage{})
	app.Route("/users", &UsersPage{})

	// That's it. RyxoGo handles everything else.
	app.Run()
}
