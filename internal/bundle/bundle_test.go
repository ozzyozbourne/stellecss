package bundle

import (
	"strings"
	"testing"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

// Compact output must keep every custom property while being smaller, and
// readable output must keep the section markers people navigate by.
func TestCompactPresetShrinksOutputWithoutLosingProperties(t *testing.T) {
	readable, err := Build(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	c := config.Default()
	c.Bundle.MinifyWhitespace, c.Bundle.MinifySyntax = true, true
	compact, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readable.CSS), "/* stellar:colors-theme.css */") {
		t.Error("readable bundle lost its section markers")
	}
	if compact.RawSize >= readable.RawSize || compact.BrotliSize > readable.BrotliSize {
		t.Errorf("compact %d/%d not smaller than readable %d/%d", compact.RawSize, compact.BrotliSize, readable.RawSize, readable.BrotliSize)
	}
	if compact.Properties != readable.Properties {
		t.Errorf("compact has %d properties, readable %d", compact.Properties, readable.Properties)
	}
	if compact.Hash == readable.Hash {
		t.Error("different bytes share a hash")
	}
}
