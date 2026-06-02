// Example RyxoGo app
package main

import (
	"strconv"

	rx "github.com/ahmad-nexarapp/ryxogo"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CounterPage — signals, computed, events
type CounterPage struct {
	rx.Page
	count   *signal.Signal[int]
	doubled *signal.Computed[int]
	name    *signal.Signal[string]
}

func (p *CounterPage) Setup() {
	p.count   = rx.Use(0)
	p.name    = rx.Use("world")
	p.doubled = rx.Computed(func() int { return p.count.Val() * 2 })
}

func (p *CounterPage) Render() *rx.Node {
	return rx.Div(rx.Props{Class: "p-8 max-w-md mx-auto"},
		rx.H1(rx.Props{Class: "text-2xl font-bold mb-6"},
			rx.Text("Hello, "+p.name.Val()+"!"),
		),
		rx.Input(rx.Props{
			Class:       "border rounded px-3 py-2 w-full mb-4",
			Value:       p.name.Val(),
			Placeholder: "Enter your name",
			OnInput:     func(v string) { p.name.Set(v) },
		}),
		rx.P(rx.Props{}, rx.Text("Count: "+strconv.Itoa(p.count.Val()))),
		rx.P(rx.Props{}, rx.Text("Doubled: "+strconv.Itoa(p.doubled.Val()))),
		rx.Div(rx.Props{Class: "flex gap-3"},
			rx.Button(rx.Props{
				Class:   "px-4 py-2 bg-blue-600 text-white rounded",
				OnClick: func() { p.count.Set(p.count.Val() + 1) },
			}, rx.Text("+")),
			rx.Button(rx.Props{
				Class:   "px-4 py-2 bg-gray-200 rounded",
				OnClick: func() { p.count.Set(0) },
			}, rx.Text("Reset")),
		),
	)
}

// UsersPage — async data fetching
type UsersPage struct {
	rx.Page
	users *signal.AsyncSignal[[]User]
}

func (p *UsersPage) Setup() {
	p.users = rx.Async(func() ([]User, error) {
		return rx.Get[[]User]("https://jsonplaceholder.typicode.com/users")
	})
}

func (p *UsersPage) Render() *rx.Node {
	if p.users.IsLoading() {
		return rx.Div(rx.Props{}, rx.Text("Loading..."))
	}
	if p.users.IsError() {
		return rx.Div(rx.Props{}, rx.Text("Error: "+p.users.Err().Error()))
	}

	users := p.users.Data()
	userNodes := rx.Each(users, func(u User, _ int) *rx.Node {
		return rx.Div(rx.Props{Class: "p-4 border rounded-lg", Key: strconv.Itoa(u.ID)},
			rx.P(rx.Props{Class: "font-medium"}, rx.Text(u.Name)),
			rx.P(rx.Props{Class: "text-sm text-gray-500"}, rx.Text(u.Email)),
		)
	})

	wrapper := rx.Div(rx.Props{Class: "p-8"})
	wrapper.Children = append(wrapper.Children, rx.H1(rx.Props{}, rx.Text("Users")))
	wrapper.Children = append(wrapper.Children, userNodes...)
	return wrapper
}

func main() {
	app := rx.New()
	app.Route("/", func() rx.Component { return &CounterPage{} })
	app.Route("/users", func() rx.Component { return &UsersPage{} })
	app.Run()
}
