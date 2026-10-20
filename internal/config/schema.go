package config

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// Schema returns the JSON Schema (draft 2020-12) for [Config], derived from
// its struct tags so the schema cannot drift from the Go types.
func Schema() []byte {
	s := schemaFor(reflect.TypeFor[Config](), reflect.StructField{})
	s["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	s["$id"] = "https://github.com/ozzyozbourne/stellecss/stellar.config.schema.json"
	s["title"] = "Stellar CSS configuration"
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}

func schemaFor(t reflect.Type, f reflect.StructField) map[string]any {
	s := map[string]any{}
	if d := f.Tag.Get("desc"); d != "" {
		s["description"] = d
	}
	switch t.Kind() {
	case reflect.Struct:
		props := map[string]any{}
		var required []string
		for i := range t.NumField() {
			sf := t.Field(i)
			name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}
			props[name] = schemaFor(sf.Type, sf)
			if name != "$schema" {
				required = append(required, name)
			}
		}
		s["type"] = "object"
		s["properties"] = props
		s["required"] = required
		s["additionalProperties"] = false
	case reflect.Slice:
		s["type"] = "array"
		s["items"] = schemaFor(t.Elem(), reflect.StructField{})
		if m := f.Tag.Get("max"); m != "" {
			s["maxItems"], _ = strconv.Atoi(m)
		}
	case reflect.String:
		s["type"] = "string"
		if e := f.Tag.Get("enum"); e != "" {
			s["enum"] = strings.Split(e, "|")
		}
	case reflect.Bool:
		s["type"] = "boolean"
	case reflect.Int:
		s["type"] = "integer"
		bounds(s, f)
	case reflect.Float64:
		s["type"] = "number"
		bounds(s, f)
	default:
		panic("config: no schema mapping for " + t.String())
	}
	return s
}

func bounds(s map[string]any, f reflect.StructField) {
	for tag, key := range map[string]string{"min": "minimum", "max": "maximum"} {
		if v := f.Tag.Get(tag); v != "" {
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				panic("config: bad " + tag + " tag on " + f.Name)
			}
			s[key] = n
		}
	}
}
