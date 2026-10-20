package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The schema shipped next to stellar.config.json must accept the default
// config and reject out-of-range values, or editors would mislead users.
func TestSchemaAcceptsDefaultsAndRejectsOutOfRange(t *testing.T) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(Schema()))
	if err != nil {
		t.Fatal(err)
	}
	comp := jsonschema.NewCompiler()
	if err := comp.AddResource("schema.json", doc); err != nil {
		t.Fatal(err)
	}
	sch, err := comp.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := jsonschema.UnmarshalJSON(bytes.NewReader(Default().Marshal()))
	if err := sch.Validate(valid); err != nil {
		t.Errorf("default config rejected: %v", err)
	}
	c := Default()
	c.General.Viewport.BaseFontSize = 200
	bad, _ := jsonschema.UnmarshalJSON(bytes.NewReader(c.Marshal()))
	if sch.Validate(bad) == nil {
		t.Error("baseFontSize 200 accepted")
	}
}

func TestParseRejectsUnknownFieldsAndKeepsDefaultsForMissingOnes(t *testing.T) {
	if _, err := Parse([]byte(`{"version":1,"genral":{}}`)); err == nil || !strings.Contains(err.Error(), "genral") {
		t.Errorf("typo key not rejected: %v", err)
	}
	c, err := Parse([]byte(`{"version":1,"general":{"viewport":{"min":360,"max":1600,"baseFontSize":16}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.General.Viewport.Min != 360 || c.Colors.Theme.Steps != Default().Colors.Theme.Steps {
		t.Errorf("got viewport.min %v, theme.steps %v", c.General.Viewport.Min, c.Colors.Theme.Steps)
	}
	b, _ := json.Marshal(c)
	if !json.Valid(b) {
		t.Error("round trip produced invalid JSON")
	}
}
