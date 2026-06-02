# Performance & Deployment

An honest guide to RyxoGo's performance characteristics and how to deploy
for the best real-world results.

## The size reality

Go compiles to a single WebAssembly binary that includes the Go runtime.
This means a baseline cost that JavaScript frameworks don't have:

| Build | Raw | Gzipped | Brotli |
|-------|-----|---------|--------|
| `rxgo build` (standard Go) | ~2.3 MB | ~650 KB | ~550 KB |
| `rxgo build --prod` (TinyGo) | ~500 KB | ~180 KB | ~140 KB |
| React (for comparison) | 180 KB | 60 KB | — |
| Svelte (for comparison) | 30 KB | 10 KB | — |

**We will not beat Svelte or React on first-load bytes.** That's structural —
Go ships a runtime, Svelte ships almost nothing. If first-visit bandwidth on
a marketing site is your top priority, use a JS framework. RyxoGo's strengths
are elsewhere: Go type safety, compiled runtime speed, and a single language
across your stack.

## What actually makes RyxoGo fast in production

The naive load test — serving WASM off one origin with no caching — measures
the worst case nobody ships. Three things change the picture entirely:

### 1. Build small: `rxgo build --prod`

Uses TinyGo (~4x smaller binary) + content hashing + brotli. Requires
[TinyGo installed](https://tinygo.org/getting-started/install/). Falls back
to standard Go automatically if TinyGo isn't present.

```bash
rxgo build --prod
# app.<hash>.wasm  ~500 KB  (~140 KB brotli)
```

### 2. Cache forever: content-hashed filenames

`--hash` (included in `--prod`) names the binary `app.<hash>.wasm`. Because
the name changes only when the content changes, you can cache it **forever**:

```
Cache-Control: public, max-age=31536000, immutable
```

This is the key insight the cold-load benchmark misses: **a user pays the
download once.** Every subsequent visit — and every other page in the app —
costs **zero bytes** for the binary. The "1000 cold visits" scenario only
happens if all 1000 users are first-time visitors with empty caches, which
is not how real traffic works.

rxgo writes the cache headers for you (`_headers` for Netlify, a
`nginx.conf.sample` for nginx).

### 3. Serve from a CDN

A static `dist/` folder belongs on a CDN (Cloudflare Pages, Netlify, Vercel,
S3+CloudFront), not a single VPS. The CDN serves the precompressed `.br`/`.gz`
from edge nodes near the user. The origin-server RPS number becomes irrelevant
because the origin barely gets hit.

## The honest comparison

| Scenario | Verdict |
|----------|---------|
| First visit, fast connection | JS wins by a few hundred ms |
| First visit, slow 3G mobile | JS wins clearly — avoid RyxoGo here |
| Repeat visits (cached) | **Tie** — both load from cache |
| Runtime interactions | **RyxoGo wins** — compiled, no JIT warmup |
| Compute-heavy work | **RyxoGo wins 2-4x** — data grids, parsing, math |
| Long sessions | **RyxoGo wins** — stable memory, no GC stutter |
| SEO / marketing | Use SSR (`ssr` package) or a JS framework |

## Best fit

RyxoGo is the right tool for **internal tools, dashboards, admin panels, and
data-heavy apps** where users load once per session and then interact heavily.
The first-load tax is paid once and the runtime advantages compound for the
rest of the session.

It is the wrong tool for marketing sites, SEO-critical pages, and apps
targeting slow mobile networks with mostly one-time visitors.

## Quick deploy checklist

1. `rxgo build --prod`
2. Upload `dist/` to a CDN-backed static host
3. Confirm `.wasm` is served as `application/wasm`
4. Confirm hashed assets get `Cache-Control: immutable`
5. Confirm SPA fallback routes unknown paths to `index.html`

The generated `_headers`, `_redirects`, and `nginx.conf.sample` handle 3–5
for you on supported hosts.
