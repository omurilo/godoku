package openapi

import (
	"encoding/json"
	"strings"
)

// GenerateExampleValue builds an example value for a schema. An explicit
// `example` on the schema always wins; otherwise it composes from properties /
// items / enum, falling back to type-based placeholders.
func GenerateExampleValue(schema *Schema, name string) interface{} {
	if schema == nil {
		return "example"
	}

	// 1) Explicit example on the schema wins (any type).
	if schema.Example != nil {
		return schema.Example
	}

	// 2) Composition keywords (allOf merges objects; one/anyOf picks the first).
	if len(schema.Properties) == 0 {
		if len(schema.AllOf) > 0 {
			merged := map[string]interface{}{}
			for _, sub := range schema.AllOf {
				if v, ok := GenerateExampleValue(sub, name).(map[string]interface{}); ok {
					for k, val := range v {
						merged[k] = val
					}
				}
			}
			if len(merged) > 0 {
				return merged
			}
		}
		if schema.Type == "" {
			if len(schema.OneOf) > 0 {
				return GenerateExampleValue(schema.OneOf[0], name)
			}
			if len(schema.AnyOf) > 0 {
				return GenerateExampleValue(schema.AnyOf[0], name)
			}
		}
	}

	// 3) Structural types.
	switch schema.Type {
	case "object":
		return objectExample(schema)
	case "array":
		if schema.Items != nil {
			return []interface{}{GenerateExampleValue(schema.Items, name)}
		}
		return []interface{}{}
	}

	// 4) Enum: first value.
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	// 5) Type-based placeholders.
	switch schema.Type {
	case "string":
		lname := strings.ToLower(name)
		switch {
		case schema.Format == "date-time" || strings.Contains(lname, "date"):
			return "2023-12-31T23:59:59Z"
		case schema.Format == "email" || strings.Contains(lname, "email"):
			return "user@example.com"
		case schema.Format == "uuid":
			return "123e4567-e89b-12d3-a456-426614174000"
		case strings.Contains(lname, "id"):
			return "123"
		default:
			return name + "_example"
		}
	case "integer", "number":
		return 42
	case "boolean":
		return true
	}

	// 6) Untyped object with properties.
	if len(schema.Properties) > 0 {
		return objectExample(schema)
	}
	return "example"
}

func objectExample(schema *Schema) map[string]interface{} {
	m := map[string]interface{}{}
	for k, v := range schema.Properties {
		m[k] = GenerateExampleValue(v, k)
	}
	return m
}

// GenerateExampleObject renders a pretty-printed JSON example for a schema.
func GenerateExampleObject(schema *Schema) string {
	if schema == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(GenerateExampleValue(schema, "root"), "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
