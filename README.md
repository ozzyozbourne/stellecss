# stellecss

A Go clone of [Stellar CSS](https://data-star.dev), the Datastar Pro design-token
editor demoed by Delaney Gillilan and Ben Croker in *Stellar CSS and internalising
the Tao*. Stellar's source is not public; this rebuild follows the podcast demo
and its screenshots.

`stellar.config.json` is the single source of truth. Every section of the editor
edits it, every edit regenerates the CSS custom properties, and every open page
that listens on `/live-refresh` swaps in the new stylesheet without reloading.

```sh
go run ./cmd/stellar init                 # write stellar.config.json and its schema
go run ./cmd/stellar build                # write bundle.output (css/stellar.css)
go run ./cmd/stellar serve -addr :7331    # run the editor
go run ./cmd/stellar schema               # print the JSON Schema
```

Requires Go 1.27.3.

## Layout

| Package | Role |
| --- | --- |
| `internal/config` | Config types, defaults (kitchen sink: everything on), load and validate, JSON Schema generated from struct tags. |
| `internal/color` | sRGB ⇄ OKLab ⇄ OKLCH, gamut mapping, APCA contrast and `-on`/`-dim` text colours, k-means chip extraction from images. |
| `internal/gen` | One generator per section, emitting `/* stellar:<section>.css */` blocks. |
| `internal/bundle` | esbuild (Go API) minification, brotli size, custom-property count, content hash. |
| `internal/web` | The Datastar editor, `/assets/css/stellar/{hash}`, `/live-refresh` and the PITA demo at `/examples/pita`. |
| `cmd/stellar` | CLI. |

## Sections

- **General:** viewport, fluid size scale (`--size-N: clamp(...)` from base min/max rem and a musical ratio at each end), z-index layers, aspect ratios, normalize.
- **Colors:** theme (image chips, harmony modes, ranked mini palettes, eased tone ramps, APCA `-on`/`-dim` targets, dark mode as `prefers-color-scheme` + `[data-theme]` or `light-dark()`), named brand and language colours with relative steps, chart qualitative and diverging palettes, code syntax palettes, gradients.
- **Fonts:** families (Modern Font Stacks plus 12 UI-library presets), imports, weights, fluid sizes, letter spacing, line heights.
- **Surface and motion:** radius, border widths, layered shadows, durations and easings, with optional reduced-motion overrides.
- **Export:** readable/compact presets, output path, namespace, colour format (`oklch()` or `#rrggbb`), colour fallbacks, font imports, esbuild minify flags, source maps and line limit, with live size stats.

## Live refresh

Any page can follow the editor:

```html
<link id="stellar-css" rel="stylesheet" href="/assets/css/stellar/<hash>">
<body data-init="@get('/live-refresh', {openWhenHidden: true})">
```

The stream sends a `datastar-patch-elements` event with a new `<link id="stellar-css">`
after every change; Datastar morphs the `href` and the browser restyles.

## Differences from the real Stellar

These are reconstructions or gaps, not copies:

- Palette ranking, colour names, code-variant scoring and chart hue selection are our own heuristics; real Stellar's algorithms are unknown, so output values differ.
- The harmony map is a static SVG wheel; Stellar's interactive wheel with a draggable easing curve is not reproduced.
- Templates use `html/template` instead of templ, and there is no scoped-CSS or per-subsection state store as in Stellar's `internal/web` tree.
- Fonts → import has no font browser; it is a list of stylesheet URLs.
- Surface and Motion sections are inferred from the icon bar; their exact contents in Stellar are not visible in the demo.
- No Datastar Pro features (no Rocket components, no `data-persist`), no auth and no multi-project support.
