package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	OpenAPI    string               `json:"openapi" yaml:"openapi"`
	Info       Info                 `json:"info" yaml:"info"`
	Servers    []Server             `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths" yaml:"paths"`
	Tags       []Tag                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Components *Components          `json:"components,omitempty" yaml:"components,omitempty"`
}

type Components struct {
	Schemas       map[string]*Schema      `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Parameters    map[string]*Parameter   `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBodies map[string]*RequestBody `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Responses     map[string]*Response    `json:"responses,omitempty" yaml:"responses,omitempty"`
}

type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	Head    *Operation `json:"head,omitempty" yaml:"head,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security    []SecurityReq       `json:"security,omitempty" yaml:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Schema struct {
	Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Required    []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example     interface{}        `json:"example,omitempty" yaml:"example,omitempty"`
	AllOf       []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf       []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf       []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	RefName     string             `json:"-" yaml:"-"`
}

type SecurityReq map[string][]string

type Endpoint struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   map[string]Response
	Deprecated  bool
	Slug        string
}

type APIDoc struct {
	Slug        string
	Title       string
	Description string
	Version     string
	Servers     []Server
	Tags        []Tag
	Endpoints   []Endpoint
	TagGroups   map[string][]Endpoint
}

func LoadSpec(filePath string) (*Spec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var spec Spec
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing JSON spec: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing YAML spec: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported spec format: %s", ext)
	}

	return &spec, nil
}

func ParseSpec(spec *Spec) *APIDoc {
	// Resolve all $ref references before building the doc
	resolveAllRefs(spec)

	doc := &APIDoc{
		Title:       spec.Info.Title,
		Description: spec.Info.Description,
		Version:     spec.Info.Version,
		Servers:     spec.Servers,
		Tags:        spec.Tags,
		TagGroups:   make(map[string][]Endpoint),
	}

	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := spec.Paths[path]
		methods := []struct {
			name string
			op   *Operation
		}{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
			{"OPTIONS", item.Options},
			{"HEAD", item.Head},
		}

		for _, m := range methods {
			if m.op == nil {
				continue
			}

			slug := buildSlug(m.name, path)
			endpoint := Endpoint{
				Method:      m.name,
				Path:        path,
				Summary:     m.op.Summary,
				Description: m.op.Description,
				Tags:        m.op.Tags,
				Parameters:  m.op.Parameters,
				RequestBody: m.op.RequestBody,
				Responses:   m.op.Responses,
				Deprecated:  m.op.Deprecated,
				Slug:        slug,
			}

			doc.Endpoints = append(doc.Endpoints, endpoint)

			if len(endpoint.Tags) == 0 {
				doc.TagGroups["default"] = append(doc.TagGroups["default"], endpoint)
			} else {
				for _, tag := range endpoint.Tags {
					doc.TagGroups[tag] = append(doc.TagGroups[tag], endpoint)
				}
			}
		}
	}

	return doc
}

func buildSlug(method, path string) string {
	slug := strings.ToLower(method) + "-" + path
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "{", "")
	slug = strings.ReplaceAll(slug, "}", "")
	slug = strings.Trim(slug, "-")
	return slug
}

func (s *Schema) TypeString() string {
	if s == nil {
		return ""
	}
	if s.RefName != "" {
		return s.RefName
	}
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
		return parts[len(parts)-1]
	}
	if s.Type == "array" && s.Items != nil {
		return "array<" + s.Items.TypeString() + ">"
	}
	t := s.Type
	if s.Format != "" {
		t += " (" + s.Format + ")"
	}
	return t
}

// resolveAllRefs resolves all $ref references in the spec by inlining component definitions.
func resolveAllRefs(spec *Spec) {
	if spec.Components == nil {
		return
	}

	// First resolve refs within components themselves (nested refs)
	for _, schema := range spec.Components.Schemas {
		resolveSchemaRefs(schema, spec.Components, make(map[string]bool))
	}

	// Then resolve refs in all paths/operations
	for _, pathItem := range spec.Paths {
		ops := []*Operation{
			pathItem.Get, pathItem.Post, pathItem.Put,
			pathItem.Delete, pathItem.Patch, pathItem.Options, pathItem.Head,
		}
		for _, op := range ops {
			if op == nil {
				continue
			}
			resolveOperationRefs(op, spec.Components)
		}
	}
}

func resolveOperationRefs(op *Operation, components *Components) {
	for i := range op.Parameters {
		if op.Parameters[i].Schema != nil {
			resolveSchemaRefs(op.Parameters[i].Schema, components, make(map[string]bool))
		}
	}

	if op.RequestBody != nil {
		for ct, media := range op.RequestBody.Content {
			if media.Schema != nil {
				resolveSchemaRefs(media.Schema, components, make(map[string]bool))
				op.RequestBody.Content[ct] = media
			}
		}
	}

	for code, resp := range op.Responses {
		for ct, media := range resp.Content {
			if media.Schema != nil {
				resolveSchemaRefs(media.Schema, components, make(map[string]bool))
				resp.Content[ct] = media
			}
		}
		op.Responses[code] = resp
	}
}

func resolveSchemaRefs(schema *Schema, components *Components, seen map[string]bool) {
	if schema == nil {
		return
	}

	// Resolve $ref
	if schema.Ref != "" {
		refName := extractRefName(schema.Ref)
		if refName == "" || seen[refName] {
			return
		}
		seen[refName] = true

		if components.Schemas != nil {
			if resolved, ok := components.Schemas[refName]; ok {
				// Recursively resolve the component schema first
				resolveSchemaRefs(resolved, components, seen)
				// Inline the resolved schema, keeping the ref name for display
				schema.RefName = refName
				schema.Ref = ""
				if schema.Type == "" {
					schema.Type = resolved.Type
				}
				if schema.Description == "" {
					schema.Description = resolved.Description
				}
				if schema.Properties == nil {
					schema.Properties = resolved.Properties
				}
				if schema.Items == nil {
					schema.Items = resolved.Items
				}
				if schema.Required == nil {
					schema.Required = resolved.Required
				}
				if schema.Enum == nil {
					schema.Enum = resolved.Enum
				}
				if schema.Format == "" {
					schema.Format = resolved.Format
				}
				if schema.AllOf == nil {
					schema.AllOf = resolved.AllOf
				}
				if schema.OneOf == nil {
					schema.OneOf = resolved.OneOf
				}
				if schema.AnyOf == nil {
					schema.AnyOf = resolved.AnyOf
				}
			}
		}
		delete(seen, refName)
	}

	// Resolve allOf by merging properties
	if len(schema.AllOf) > 0 {
		for _, sub := range schema.AllOf {
			resolveSchemaRefs(sub, components, seen)
		}
		merged := mergeAllOf(schema.AllOf)
		if schema.Type == "" {
			schema.Type = merged.Type
		}
		if schema.Properties == nil {
			schema.Properties = merged.Properties
		} else {
			for k, v := range merged.Properties {
				if _, exists := schema.Properties[k]; !exists {
					schema.Properties[k] = v
				}
			}
		}
		if schema.Required == nil {
			schema.Required = merged.Required
		}
		schema.AllOf = nil
	}

	// Resolve nested
	for _, prop := range schema.Properties {
		resolveSchemaRefs(prop, components, seen)
	}
	if schema.Items != nil {
		resolveSchemaRefs(schema.Items, components, seen)
	}
	for _, sub := range schema.OneOf {
		resolveSchemaRefs(sub, components, seen)
	}
	for _, sub := range schema.AnyOf {
		resolveSchemaRefs(sub, components, seen)
	}
}

func extractRefName(ref string) string {
	// #/components/schemas/MyModel -> MyModel
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func mergeAllOf(schemas []*Schema) *Schema {
	merged := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}
	for _, s := range schemas {
		if s.Type != "" {
			merged.Type = s.Type
		}
		for k, v := range s.Properties {
			merged.Properties[k] = v
		}
		merged.Required = append(merged.Required, s.Required...)
	}
	return merged
}

// DiscoverAPIs scans the apis/ directory in rootDir for OpenAPI spec files (.yaml, .yml, .json).
func DiscoverAPIs(rootDir string) []string {
	apisDir := filepath.Join(rootDir, "apis")
	entries, err := os.ReadDir(apisDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			files = append(files, filepath.Join("apis", entry.Name()))
		}
	}
	return files
}

func LoadAllSpecs(files []string, rootDir string) ([]*APIDoc, error) {
	var docs []*APIDoc

	for _, file := range files {
		path := file
		if !filepath.IsAbs(path) {
			path = filepath.Join(rootDir, path)
		}

		spec, err := LoadSpec(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", file, err)
		}

		base := filepath.Base(file)
		slug := strings.TrimSuffix(base, filepath.Ext(base))

		doc := ParseSpec(spec)
		doc.Slug = slug
		docs = append(docs, doc)
	}

	return docs, nil
}
