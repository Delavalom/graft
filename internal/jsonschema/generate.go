package jsonschema

import (
	"encoding/json"
	"reflect"
)

func Generate[T any]() (json.RawMessage, error) {
	var zero T
	t := reflect.TypeOf(zero)
	schema := generateType(t)
	return json.Marshal(schema)
}

func GenerateFromType(t reflect.Type) (json.RawMessage, error) {
	schema := generateType(t)
	return json.Marshal(schema)
}

func generateType(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return generateObject(t)
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{
			"type":  "array",
			"items": generateType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": generateType(t.Elem()),
		}
	default:
		return map[string]any{"type": "string"}
	}
}

func generateObject(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		omitempty := false
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			parts := splitTag(tag)
			if parts[0] != "" {
				name = parts[0]
			}
			for _, p := range parts[1:] {
				if p == "omitempty" {
					omitempty = true
				}
			}
		}

		prop := generateType(field.Type)
		if desc := field.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}
		properties[name] = prop

		if !omitempty {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func splitTag(tag string) []string {
	var parts []string
	current := ""
	for _, c := range tag {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}
