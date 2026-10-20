// Package bundle turns generated CSS into the shipped stylesheet: esbuild
// minifies it according to the config and the result is measured the way the
// editor's Bundle panel reports it.
package bundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/andybalholm/brotli"
	"github.com/evanw/esbuild/pkg/api"

	"github.com/ozzyozbourne/stellecss/internal/config"
	"github.com/ozzyozbourne/stellecss/internal/gen"
)

// Bundle is a finished stylesheet and its statistics.
type Bundle struct {
	CSS []byte
	// Map is the external source map; empty unless bundle.sourcemap is "external".
	Map        []byte
	Properties int
	RawSize    int
	BrotliSize int
	// Hash is a short content hash used in cache-busting URLs.
	Hash   string
	Result *gen.Result
}

// Build generates and minifies the stylesheet for c.
func Build(c *config.Config) (*Bundle, error) {
	r := gen.Build(c)
	b := c.Bundle
	opts := api.TransformOptions{
		Loader:            api.LoaderCSS,
		Sourcefile:        "stellar.css",
		MinifyWhitespace:  b.MinifyWhitespace,
		MinifySyntax:      b.MinifySyntax,
		MinifyIdentifiers: b.MinifyIdentifiers,
		LineLimit:         b.LineLimit,
		SourceRoot:        b.SourceRoot,
		// Stellar's tokens target evergreen browsers; esbuild must not lower
		// oklch() or light-dark(), since the colour fallbacks already cover that.
		Engines: []api.Engine{{Name: api.EngineChrome, Version: "123"}, {Name: api.EngineFirefox, Version: "120"}, {Name: api.EngineSafari, Version: "17.5"}},
	}
	switch b.Sourcemap {
	case "inline":
		opts.Sourcemap = api.SourceMapInline
	case "external":
		opts.Sourcemap = api.SourceMapExternal
	}
	if b.SourcesContent {
		opts.SourcesContent = api.SourcesContentInclude
	} else {
		opts.SourcesContent = api.SourcesContentExclude
	}
	res := api.Transform(r.CSS, opts)
	if len(res.Errors) > 0 {
		var errs []error
		for _, m := range res.Errors {
			errs = append(errs, errors.New(m.Text))
		}
		return nil, fmt.Errorf("esbuild: %w", errors.Join(errs...))
	}
	out := res.Code
	// esbuild drops ordinary comments, which would lose the
	// /* stellar:<section>.css */ markers of the readable preset. With no
	// minify flag esbuild only validates, and the rendered text ships.
	if !b.MinifyWhitespace && !b.MinifySyntax && !b.MinifyIdentifiers && opts.Sourcemap == api.SourceMapNone {
		out = []byte(r.CSS)
	}
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	sum := sha256.Sum256(out)
	return &Bundle{
		CSS:        out,
		Map:        res.Map,
		Properties: gen.CountProperties(string(out)),
		RawSize:    len(out),
		BrotliSize: BrotliSize(out),
		Hash:       hex.EncodeToString(sum[:])[:16],
		Result:     r,
	}, nil
}

// BrotliSize compresses b at the quality CDNs use for static assets.
func BrotliSize(b []byte) int {
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	w.Write(b)
	w.Close()
	return buf.Len()
}

// KiB formats a byte count the way the Bundle panel does, e.g. "69.0 KiB".
func KiB(n int) string {
	return fmt.Sprintf("%.1f KiB", float64(n)/1024)
}
