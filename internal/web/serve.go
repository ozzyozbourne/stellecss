// Package web is Stellar's editor: a Datastar UI where every change rewrites
// stellar.config.json, regenerates the bundle and pushes the new stylesheet to
// every open page over /live-refresh.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/ozzyozbourne/stellecss/internal/bundle"
	"github.com/ozzyozbourne/stellecss/internal/color"
	"github.com/ozzyozbourne/stellecss/internal/config"
)

const version = "v0.0.2"

//go:embed templates static
var assets embed.FS

var tmpl = template.Must(template.New("").Funcs(funcs).ParseFS(assets, "templates/*.html", "templates/sections/*.html"))

type server struct {
	store   *store
	log     *slog.Logger
	cfgPath string
	// base is the default bundle. The editor loads it before the live
	// stylesheet so its own chrome survives sections being disabled.
	base *bundle.Bundle
}

// Serve runs the editor until ctx is cancelled.
func Serve(ctx context.Context, addr, cfgPath string, log *slog.Logger) error {
	h, err := Handler(cfgPath, log)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: h, BaseContext: func(net.Listener) context.Context { return ctx }}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Info("stellar editor", "addr", addr, "config", cfgPath)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Handler builds the editor's routes for the config at cfgPath, creating a
// default config there if none exists.
func Handler(cfgPath string, log *slog.Logger) (http.Handler, error) {
	st, err := openStore(cfgPath)
	if err != nil {
		return nil, err
	}
	base, err := bundle.Build(config.Default())
	if err != nil {
		return nil, err
	}
	s := &server{store: st, log: log, cfgPath: cfgPath, base: base}
	static, _ := fs.Sub(assets, "static")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/general/viewport", http.StatusFound)
	})
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /assets/css/stellar/{hash}", s.css)
	mux.HandleFunc("GET /assets/css/base", s.baseCSS)
	mux.HandleFunc("GET /live-refresh", s.liveRefresh)
	mux.HandleFunc("GET /examples/pita", s.pita)
	mux.HandleFunc("GET /stellar.config.json", s.download("application/json", func(c *config.Config) []byte { return c.Marshal() }))
	mux.HandleFunc("GET /stellar.config.schema.json", s.download("application/schema+json", func(*config.Config) []byte { return config.Schema() }))
	mux.HandleFunc("GET /theme-image", s.themeImage)
	for _, g := range groups {
		for _, sec := range g.Sections {
			mux.HandleFunc("GET "+sec.URL(), s.page(sec))
			mux.HandleFunc("POST "+sec.URL(), s.update(sec))
			mux.HandleFunc("POST /reset"+sec.URL(), s.resetSection(sec))
		}
	}
	mux.HandleFunc("POST /reset", s.resetAll)
	mux.HandleFunc("POST /list/{op}", s.listOp)
	mux.HandleFunc("POST /action/{name}", s.action)
	return logRequests(log, mux), nil
}

func logRequests(log *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Debug("request", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start))
	})
}

func (s *server) page(sec section) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, b := s.store.snapshot()
		s.render(w, "layout", newView(sec, c, b, s.cfgPath))
	}
}

func (s *server) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		s.log.Error("render", "template", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// panels renders the parts of the page a config change can affect.
func (s *server) panels(v *view) (string, error) {
	var buf bytes.Buffer
	for _, name := range []string{"settings", "preview"} {
		if err := tmpl.ExecuteTemplate(&buf, name, v); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// respond patches the current section's panels, clears any error and, when
// signals is true, replaces the cfg signal (after a server-side edit).
func (s *server) respond(sse *datastar.ServerSentEventGenerator, sec section, signals bool) {
	c, b := s.store.snapshot()
	v := newView(sec, c, b, s.cfgPath)
	if signals {
		if err := sse.PatchSignals([]byte(v.Signals())); err != nil {
			return
		}
	}
	html, err := s.panels(v)
	if err != nil {
		s.fail(sse, err)
		return
	}
	sse.PatchElements(html)
	sse.PatchElements(`<output id="errors" role="alert"></output>`)
}

func (s *server) fail(sse *datastar.ServerSentEventGenerator, err error) {
	s.log.Warn("rejected change", "err", err)
	sse.PatchElements(`<output id="errors" role="alert">` + template.HTMLEscapeString(err.Error()) + `</output>`)
}

type signals struct {
	Cfg map[string]any `json:"cfg"`
}

// update applies the posted config. The whole config travels in the cfg
// signal, so the server never merges partial state.
func (s *server) update(sec section) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sig signals
		if err := datastar.ReadSignals(r, &sig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sse := datastar.NewSSE(w, r)
		c, err := fromTree(sig.Cfg)
		if err == nil {
			err = s.store.apply(c)
		}
		if err != nil {
			s.fail(sse, err)
			return
		}
		s.respond(sse, sec, false)
	}
}

// edit applies a server-side change to the generic tree of the current
// config, then refreshes the posting page including its signals.
func (s *server) edit(w http.ResponseWriter, r *http.Request, sec section, f func(tree map[string]any) error) {
	sse := datastar.NewSSE(w, r)
	c, _ := s.store.snapshot()
	tree := toTree(c)
	err := f(tree)
	if err == nil {
		if c, err = fromTree(tree); err == nil {
			err = s.store.apply(c)
		}
	}
	if err != nil {
		s.fail(sse, err)
		return
	}
	s.respond(sse, sec, true)
}

func currentSection(r *http.Request) section {
	g, sl, _ := strings.Cut(strings.Trim(r.URL.Query().Get("from"), "/"), "/")
	if sec, ok := findSection(g, sl); ok {
		return sec
	}
	return groups[0].Sections[0]
}

func (s *server) resetSection(sec section) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.edit(w, r, sec, func(tree map[string]any) error {
			def, err := lookup(toTree(config.Default()), sec.Path)
			if err != nil {
				return err
			}
			return assign(tree, sec.Path, def)
		})
	}
}

func (s *server) resetAll(w http.ResponseWriter, r *http.Request) {
	s.edit(w, r, currentSection(r), func(tree map[string]any) error {
		for k, v := range toTree(config.Default()) {
			tree[k] = v
		}
		return nil
	})
}

// listOp adds, removes or moves an item of the array at ?path=.
func (s *server) listOp(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := q.Get("path")
	i, _ := strconv.Atoi(q.Get("i"))
	s.edit(w, r, currentSection(r), func(tree map[string]any) error {
		x, err := lookup(tree, path)
		if err != nil {
			return err
		}
		list, ok := x.([]any)
		if !ok {
			return fmt.Errorf("%s is not a list", path)
		}
		switch r.PathValue("op") {
		case "add":
			list = append(list, newItem(list, path))
		case "remove":
			if i < 0 || i >= len(list) {
				return fmt.Errorf("no item %d in %s", i, path)
			}
			list = slices.Delete(list, i, i+1)
		case "clear":
			list = []any{}
		default:
			return fmt.Errorf("unknown list op %q", r.PathValue("op"))
		}
		return assign(tree, path, list)
	})
}

// newItem copies the last item, or the first default item for empty lists,
// renaming it so the new row is distinguishable.
func newItem(list []any, path string) any {
	src := list
	if len(src) == 0 {
		d, err := lookup(toTree(config.Default()), path)
		if l, ok := d.([]any); err == nil && ok {
			src = l
		}
	}
	if len(src) == 0 {
		if strings.HasSuffix(path, ".pairs") {
			return map[string]any{"from": 1, "to": 2}
		}
		return map[string]any{"name": "custom", "value": 0}
	}
	item := cloneJSON(src[len(src)-1])
	if m, ok := item.(map[string]any); ok {
		if n, ok := m["name"].(string); ok {
			m["name"] = fmt.Sprintf("%s-%03d", strings.TrimRightFunc(n, func(r rune) bool { return r == '-' || (r >= '0' && r <= '9') }), len(list)+1)
		}
	}
	return item
}

func cloneJSON(x any) any {
	switch t := x.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, v := range t {
			m[k] = cloneJSON(v)
		}
		return m
	case []any:
		l := make([]any, len(t))
		for i, v := range t {
			l[i] = cloneJSON(v)
		}
		return l
	}
	return x
}

// action runs the named panel buttons that are not plain field edits.
func (s *server) action(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	q := r.URL.Query()
	sec := currentSection(r)
	switch name {
	case "upsert-brands", "upsert-languages", "sort-named", "palette", "code-variant", "font-preset", "bundle-preset":
		s.edit(w, r, sec, func(tree map[string]any) error {
			c, err := fromTree(tree)
			if err != nil {
				return err
			}
			if err := applyAction(c, name, q.Get("v")); err != nil {
				return err
			}
			for k, v := range toTree(c) {
				tree[k] = v
			}
			return nil
		})
	case "theme-image":
		s.uploadImage(w, r, sec)
	default:
		http.NotFound(w, r)
	}
}

func applyAction(c *config.Config, name, arg string) error {
	switch name {
	case "upsert-brands", "upsert-languages":
		src := config.BrandColors
		if name == "upsert-languages" {
			src = config.LanguageColors
		}
		for _, n := range src {
			i := slices.IndexFunc(c.Colors.Named.Tokens, func(t config.NamedColor) bool { return t.Name == n.Name })
			if i >= 0 {
				c.Colors.Named.Tokens[i] = n
			} else {
				c.Colors.Named.Tokens = append(c.Colors.Named.Tokens, n)
			}
		}
	case "sort-named":
		slices.SortFunc(c.Colors.Named.Tokens, func(a, b config.NamedColor) int { return strings.Compare(a.Name, b.Name) })
	case "palette":
		i, err := strconv.Atoi(arg)
		if err != nil {
			return err
		}
		c.Colors.Theme.Palette = i
	case "code-variant":
		i, err := strconv.Atoi(arg)
		if err != nil {
			return err
		}
		c.Colors.Code.Variant = i
	case "font-preset":
		p, ok := config.FontPresets[arg]
		if !ok {
			return fmt.Errorf("unknown font preset %q", arg)
		}
		for i, f := range c.Fonts.Families.Families {
			if st, ok := p[f.Name]; ok {
				c.Fonts.Families.Families[i].Stack = slices.Clone(st)
			}
		}
		c.Fonts.Families.Preset = arg
	case "bundle-preset":
		b := &c.Bundle
		compact := arg == "compact"
		b.MinifyWhitespace, b.MinifySyntax, b.MinifyIdentifiers = compact, compact, false
		b.ColorFallbacks = !compact
		b.Sourcemap = "none"
		b.LineLimit = 0
	}
	return nil
}

// uploadImage extracts theme seeds from the image in the themeImage signal,
// stores the image next to the config and points the theme at it.
func (s *server) uploadImage(w http.ResponseWriter, r *http.Request, sec section) {
	var sig struct {
		ThemeImage []struct {
			Name, Contents, Mime string
		} `json:"themeImage"`
	}
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.edit(w, r, sec, func(tree map[string]any) error {
		if len(sig.ThemeImage) == 0 {
			return errors.New("choose an image first")
		}
		raw, err := base64.StdEncoding.DecodeString(sig.ThemeImage[0].Contents)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return fmt.Errorf("decode image: %w", err)
		}
		name := "theme" + strings.ToLower(filepath.Ext(sig.ThemeImage[0].Name))
		if err := os.WriteFile(filepath.Join(filepath.Dir(s.cfgPath), name), raw, 0o644); err != nil {
			return err
		}
		var seeds []any
		for _, ch := range color.Extract(img, 24) {
			seeds = append(seeds, ch.Hex())
		}
		th := tree["colors"].(map[string]any)["theme"].(map[string]any)
		th["image"], th["seeds"], th["palette"] = name, seeds, 0
		return nil
	})
}

func (s *server) themeImage(w http.ResponseWriter, r *http.Request) {
	c, _ := s.store.snapshot()
	if c.Colors.Theme.Image == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(filepath.Dir(s.cfgPath), filepath.Base(c.Colors.Theme.Image)))
}

func (s *server) css(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.css(r.PathValue("hash"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	// Content-addressed, so the URL changes whenever the bytes do.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(b)
}

func (s *server) baseCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(s.base.CSS)
}

func (s *server) download(mime string, body func(*config.Config) []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, _ := s.store.snapshot()
		w.Header().Set("Content-Type", mime)
		w.Write(body(c))
	}
}

// StylesheetLink is the element /live-refresh patches; pages include it with
// the same id so Datastar morphs its href in place.
func StylesheetLink(hash string) string {
	return `<link id="stellar-css" rel="stylesheet" href="/assets/css/stellar/` + hash + `">`
}

// liveRefresh holds an SSE stream open and sends the stylesheet link after
// every change. It sends the current link first, so a page that reconnects
// after missing an update catches up.
func (s *server) liveRefresh(w http.ResponseWriter, r *http.Request) {
	ch, cancel := s.store.subscribe()
	defer cancel()
	sse := datastar.NewSSE(w, r)
	_, b := s.store.snapshot()
	if err := sse.PatchElements(StylesheetLink(b.Hash)); err != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case h := <-ch:
			if err := sse.PatchElements(StylesheetLink(h)); err != nil {
				return
			}
		}
	}
}

func (s *server) pita(w http.ResponseWriter, r *http.Request) {
	_, b := s.store.snapshot()
	s.render(w, "pita", map[string]any{"Hash": b.Hash})
}
