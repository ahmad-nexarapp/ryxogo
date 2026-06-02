# Changelog

All notable changes to RyxoGo are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project follows [Semantic Versioning](https://semver.org/).

## [v0.3.2] — 2026-06-02

### Added
- **Fine-grained reactive attributes, styles, and conditional show/hide** via the new `rx.Bindings()` builder. Each binding updates only its target — `Render()` never re-runs.
  - `BindClass(fn)` — reactive `className`
  - `BindAttr(name, fn)` — reactive value of any attribute (`href`, `value`, `title`, …)
  - `BindStyle(prop, fn)` — reactive single CSS property
  - `BindShow(fn)` — toggle visibility via `display:none`
  - Chain multiple bindings on one element, then `.On(element)`
- SSR renders the initial computed values of all bindings (class, attrs, styles, and `display:none` when hidden) so first paint is correct before hydration.

### Example
```go
rx.Bindings().
    BindClass(func() string {
        if p.active.Val() { return "tab active" }
        return "tab"
    }).
    BindShow(func() bool { return p.visible.Val() }).
    On(rx.Div(rx.Props{Class: "card"}, children...))
```

## [v0.3.1] — 2026-06-02

### Added
- **Fine-grained reactive text** via `rx.BindText(fn)` — Solid.js-style surgical updates. When the signals read inside the function change, only that one text node updates; `Render()` does not re-run.
- Reactive text nodes compute their value once for SSR; the client effect resumes on hydration.

### Example
```go
// Coarse (re-runs Render, then diffs):
rx.Text(strconv.Itoa(p.count.Val()))

// Fine-grained (only this text node updates):
rx.BindText(func() string { return strconv.Itoa(p.count.Val()) })
```

## [v0.3.0] — 2026-06-02

### Added
- **Server-side rendering (SSR)** — `ssr.RenderToString(component)` renders components to HTML in pure Go. Fast first paint, full SEO. The SSR server adds Open Graph and Twitter meta tags per route.
- **Hydration** — the WASM bundle attaches event handlers to server-rendered DOM instead of wiping it. No blank flash.
- **Static site generation (SSG)** — `ssr.NewGenerator("dist")` renders every route to static `.html` files at build time.
- **i18n** — `i18n.T(key)` and `i18n.TF(key, vars)` with reactive locale switching; `SetLocale()` re-renders all components using translations.
- **Component library** — ready-made `ui.Button`, `ui.Card`, `ui.Input`, `ui.Modal`, `ui.Badge`, `ui.Alert`, `ui.Spinner`, `ui.Stack`, `ui.Row` with bundled CSS.
- **Testing utilities** — `rxtest.Render(component)` with `AssertText`, `AssertHasClass`, `AssertHasTag`, `AssertAttr`, `AssertTagCount`, and `Click(class)` event simulation — no browser required.
- **Lazy routes** — `rx.Lazy(factory)` defers `Setup()` until a route is first visited.

## [v0.2.1] — 2026-06-02

### Fixed
- **Link full-reload** — `rx.Link` now calls `preventDefault()` on `<a>` clicks; navigation no longer reloads the WASM bundle.
- **js.FuncOf memory leaks** — all event-listener functions are tracked and released when their node leaves the DOM. Long-lived apps no longer grow memory on re-render.
- **Render error recovery** — a panic in `Render()` now shows a styled error UI instead of a blank screen.

### Added
- **Base path support** — `app.BasePath("/myapp")` for subpath deploys.
- **`OnUnmount` lifecycle** — called when navigating away from a page.
- `LinkProps.Active` — class applied automatically when the link matches the current route.

## [v0.2.0] — 2026-06-02

### Added
- **Persistence** — `rx.Persist("key", default)` creates a signal backed by `localStorage`, surviving page refresh.
- **Two-way input binding** — `rx.BindString(signal, props)` replaces manual `Value` + `OnInput` boilerplate.
- **Bundle size tooling** — `rxgo build --compress` gzips the WASM output; build reports show the estimated gzipped size; `-ldflags "-s -w"` is applied automatically.

## [v0.1.9] — 2026-06-02

### Fixed
- **F1: Computed subscription leak** — computed signals clear old dependencies before each recompute; a cancellation flag prevents stale effect invocations.
- **F2: event listeners stacking** — elements are cloned (`cloneNode(false)`) before re-attaching handlers, so listeners never accumulate across re-renders.
- **F6: history warning** — same-path navigation uses `replaceState` instead of `pushState`.

### Added
- **Global store** — `rx.NewStore`, `rx.GetStore`, `rx.UpdateStore` (F3).
- **Client-side `rx.Link`** (F5) — navigation without a full page reload.
- More HTML element helpers: `Label`, `Textarea`, `Select`, `Option`, `Table`, `Thead`, `Tbody`, `Tr`, `Th`, `Td`, `Pre`, `Code`, `Hr`, `Br`, `Strong`, `Em`, `Small`, `Ol`, `Aside`.

## [v0.1.8] — 2026-06-02

### Added
- **Proper MCP server** — JSON-RPC 2.0 over stdio so Cursor and Claude Code can connect. `rxgo new` generates `.cursor/mcp.json` automatically. HTTP mode available with `--http`.

## [v0.1.6] — 2026-06-02

### Changed
- **Reactive rendering rebuilt on automatic dependency tracking** (`signal.Track`). Reading a signal inside `Render()` auto-subscribes the component; changes trigger a batched re-render via `requestAnimationFrame`. Same model as Solid.js / Vue 3.

## [v0.1.5] — 2026-06-02

### Fixed
- DOM diff now patches the correct parent (recursive patching), fixing `replaceChild` panics on nested updates.
- Text nodes get a DOM reference, so text updates reach the browser.
- Removed the stuck "dirty" flag; `requestAnimationFrame` coalesces renders.

## [v0.1.0] — 2026-06-02

### Added
- Initial release: signals, computed, async signals, virtual DOM renderer, file-based router, HTTP client, `rxgo` CLI (`new`, `serve`, `build`), WASM build pipeline.

[v0.3.2]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.3.2
[v0.3.1]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.3.1
[v0.3.0]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.3.0
[v0.2.1]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.2.1
[v0.2.0]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.2.0
[v0.1.9]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.1.9
[v0.1.8]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.1.8
[v0.1.6]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.1.6
[v0.1.5]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.1.5
[v0.1.0]: https://github.com/ahmad-nexarapp/ryxogo/releases/tag/v0.1.0
