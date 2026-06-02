// virtual.go — windowed list rendering for large datasets.
//
// rx.Each over thousands of rows builds thousands of virtual nodes every
// render and diffs them all — the cost grows linearly and gets slow past a
// few hundred rows. VirtualList renders only the rows currently in (or near)
// the viewport, with spacer elements above and below to preserve scroll
// height. This keeps the DOM and diff cost constant regardless of list size.
package ryxogo

import (
	"strconv"

	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

// VirtualListProps configures a virtualized list.
type VirtualListProps struct {
	// Total number of items in the full dataset.
	Count int
	// Fixed pixel height of each row. Virtualization needs a known row
	// height to compute which rows are visible.
	RowHeight int
	// Visible viewport height in pixels (the scroll container height).
	Height int
	// Overscan: extra rows rendered above/below the viewport to avoid
	// blank flashes while scrolling. Default 5.
	Overscan int
	// Class for the scroll container.
	Class string
	// Render produces the node for item at index i.
	Render func(i int) *core.Node
}

// VirtualList renders only the visible window of a large list.
//
//	type ListPage struct {
//	    rx.Page
//	    scrollTop *signal.Signal[int]
//	    rows      []Row
//	}
//	func (p *ListPage) Setup() { p.scrollTop = rx.Use(0) }
//	func (p *ListPage) Render() *rx.Node {
//	    return rx.VirtualList(rx.VirtualListProps{
//	        Count:     len(p.rows),
//	        RowHeight: 40,
//	        Height:    600,
//	        Render: func(i int) *rx.Node {
//	            return rx.Div(rx.Props{}, rx.Text(p.rows[i].Name))
//	        },
//	    }, p.scrollTop)
//	}
//
// The scrollTop signal is updated by the container's scroll handler; only
// the visible rows re-render as the user scrolls.
func VirtualList(props VirtualListProps, scrollTop *signal.Signal[int]) *core.Node {
	if props.Overscan == 0 {
		props.Overscan = 5
	}
	if props.RowHeight <= 0 {
		props.RowHeight = 40
	}

	top := scrollTop.Val()
	totalHeight := props.Count * props.RowHeight

	// Which rows are visible right now.
	first := top/props.RowHeight - props.Overscan
	if first < 0 {
		first = 0
	}
	visibleCount := props.Height/props.RowHeight + props.Overscan*2
	last := first + visibleCount
	if last > props.Count {
		last = props.Count
	}

	// Spacer above keeps scroll position correct; rendered rows; spacer below.
	offsetTop := first * props.RowHeight
	offsetBottom := totalHeight - last*props.RowHeight
	if offsetBottom < 0 {
		offsetBottom = 0
	}

	children := make([]*core.Node, 0, (last-first)+2)
	// Top spacer
	children = append(children, core.Div(core.Props{
		Style: map[string]string{"height": strconv.Itoa(offsetTop) + "px"},
	}))
	// Visible rows
	for i := first; i < last; i++ {
		row := props.Render(i)
		children = append(children, row)
	}
	// Bottom spacer
	children = append(children, core.Div(core.Props{
		Style: map[string]string{"height": strconv.Itoa(offsetBottom) + "px"},
	}))

	// Scroll container — onScroll updates the signal, re-rendering the window.
	return core.El("div", core.Props{
		Class: props.Class,
		Style: map[string]string{
			"height":     strconv.Itoa(props.Height) + "px",
			"overflow-y": "auto",
			"position":   "relative",
		},
		OnScrollTop: func(t int) { scrollTop.Set(t) },
	}, children...)
}
