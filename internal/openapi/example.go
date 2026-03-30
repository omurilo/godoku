package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

func GenerateExampleValue(schema *Schema, name string) interface{} {
	if schema == nil {
		return "example"
	}
	if examples, ok := schema.Example.([]interface{}); ok {
		if len(examples) > 0 {
			return examples[0]
		}
	}
	if schema.Example != nil {
		switch schema.Type {
		case "array":
			if arr, ok := schema.Example.([]interface{}); ok {
				return arr
			}
			if schema.Items != nil {
				return []interface{}{GenerateExampleValue(schema.Items, "item1"), GenerateExampleValue(schema.Items, "item2")}
			}
			return []interface{}{"default_array_value"}
		case "object":
			if m, ok := schema.Example.(map[string]interface{}); ok {
				return m
			}
			if len(schema.Properties) > 0 {
				m := map[string]interface{}{}
				for k, v := range schema.Properties {
					m[k] = GenerateExampleValue(v, k)
				}
				return m
			}
			return map[string]interface{}{"default_key": "default_value"}
		default:
			if schema.Type == "string" {
				return "default_string_value"
			}
			return fmt.Sprintf("%v", schema.Example)
		}
	}
	if len(schema.Enum) > 0 {
		switch schema.Type {
		case "array":
			if arr, ok := schema.Enum[0].([]interface{}); ok {
				return arr
			}
			if schema.Items != nil {
				return []interface{}{GenerateExampleValue(schema.Items, "enum_item1"), GenerateExampleValue(schema.Items, "enum_item2")}
			}
			return []interface{}{"default_enum_array_value"}
		case "object":
			if m, ok := schema.Enum[0].(map[string]interface{}); ok {
				return m
			}
			if len(schema.Properties) > 0 {
				m := map[string]interface{}{}
				for k, v := range schema.Properties {
					m[k] = GenerateExampleValue(v, k)
				}
				return m
			}
			return map[string]interface{}{"default_enum_key": "default_enum_value"}
		default:
			if schema.Type == "string" {
				return "default_enum_string_value"
			}
			return fmt.Sprintf("%v", schema.Enum[0])
		}
	}
	switch schema.Type {
	case "string":
		if strings.Contains(strings.ToLower(name), "email") {
			return "user@example.com"
		}
		if strings.Contains(strings.ToLower(name), "id") {
			return "123"
		}
		if strings.Contains(strings.ToLower(name), "date") {
			return "2023-12-31T23:59:59Z"
		}
		return name + "_example"
	case "integer", "number":
		return 42
	case "boolean":
		return true
	case "array":
		var arr []interface{}
		if schema.Items != nil {
			arr = append(arr, GenerateExampleValue(schema.Items, name))
			arr = append(arr, GenerateExampleValue(schema.Items, name+"2"))
		}
		return arr
	case "object":
		if len(schema.Properties) > 0 {
			m := map[string]interface{}{}
			for k, v := range schema.Properties {
				m[k] = GenerateExampleValue(v, k)
			}
			return m
		}
		return map[string]interface{}{}
	}
	return "example"
}

func GenerateExampleObject(schema *Schema) string {
	if schema == nil {
		return "{}"
	}
	if schema.Example != nil {
		b, _ := json.MarshalIndent(schema.Example, "", "  ")
		return string(b)
	}
	var val interface{}
	if schema.Type == "object" && len(schema.Properties) > 0 {
		m := map[string]interface{}{}
		for k, v := range schema.Properties {
			m[k] = GenerateExampleValue(v, k)
		}
		val = m
	} else {
		val = GenerateExampleValue(schema, "root")
	}
	b, _ := json.MarshalIndent(val, "", "  ")
	return string(b)
}
