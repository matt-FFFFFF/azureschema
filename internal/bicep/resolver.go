package bicep

import (
	"fmt"
	"sort"
)

// ResolvedType is the output of type resolution - a recursive structure
// that mirrors the JSON output format from the bash script.
type ResolvedType struct {
	Type        string                   `json:"type"`
	Name        string                   `json:"name,omitempty"`
	Const       string                   `json:"const,omitempty"`
	Description string                   `json:"description,omitempty"`
	Required    *bool                    `json:"required,omitempty"`
	ReadOnly    *bool                    `json:"readOnly,omitempty"`
	WriteOnly   *bool                    `json:"writeOnly,omitempty"`
	Properties  map[string]*ResolvedType `json:"properties,omitempty"`
	Items       *ResolvedType            `json:"items,omitempty"`
	OneOf       []*ResolvedType          `json:"oneOf,omitempty"`
	MinLength   *int                     `json:"minLength,omitempty"`
	MaxLength   *int                     `json:"maxLength,omitempty"`
	Pattern     *string                  `json:"pattern,omitempty"`
	Minimum     *int64                   `json:"minimum,omitempty"`
	Maximum     *int64                   `json:"maximum,omitempty"`
	Truncated   string                   `json:"_truncated,omitempty"`
}

// Resolver resolves bicep type references into a structured representation.
type Resolver struct {
	Types    TypesFile
	MaxDepth int
}

// NewResolver creates a new type resolver.
func NewResolver(types TypesFile, maxDepth int) *Resolver {
	return &Resolver{
		Types:    types,
		MaxDepth: maxDepth,
	}
}

// ResolveResourceType resolves a ResourceType entry at the given index.
// It follows the body.$ref to the ObjectType and resolves it.
func (r *Resolver) ResolveResourceType(typeIndex int) (*ResolvedType, error) {
	if typeIndex < 0 || typeIndex >= len(r.Types) {
		return nil, fmt.Errorf("type index %d out of range (0-%d)", typeIndex, len(r.Types)-1)
	}

	rt := r.Types[typeIndex]
	if rt.Type != "ResourceType" {
		return nil, fmt.Errorf("expected ResourceType at index %d, got %s", typeIndex, rt.Type)
	}

	if rt.Body == nil {
		return nil, fmt.Errorf("ResourceType at index %d has no body ref", typeIndex)
	}

	bodyIdx, err := rt.Body.Index()
	if err != nil {
		return nil, fmt.Errorf("invalid body ref in ResourceType: %w", err)
	}

	if bodyIdx < 0 || bodyIdx >= len(r.Types) {
		return nil, fmt.Errorf("body index %d out of range", bodyIdx)
	}

	return r.resolve(&r.Types[bodyIdx], 0), nil
}

// resolve recursively resolves a type entry into a ResolvedType.
func (r *Resolver) resolve(t *TypeEntry, depth int) *ResolvedType {
	if depth > r.MaxDepth {
		res := &ResolvedType{Truncated: "depth limit exceeded"}
		if t.Type == "ObjectType" || t.Type == "DiscriminatedObjectType" {
			res.Type = "object"
			if t.Name != nil {
				res.Name = *t.Name
			}
		} else {
			res.Type = typeStr(t.Type)
		}
		return res
	}

	switch t.Type {
	case "StringType":
		res := &ResolvedType{Type: "string"}
		res.MinLength = t.MinLength
		res.MaxLength = t.MaxLength
		res.Pattern = t.Pattern
		return res

	case "StringLiteralType":
		res := &ResolvedType{Type: "string"}
		if t.Value != nil {
			res.Const = *t.Value
		}
		return res

	case "IntegerType":
		res := &ResolvedType{Type: "integer"}
		res.Minimum = t.MinValue
		res.Maximum = t.MaxValue
		return res

	case "BooleanType":
		return &ResolvedType{Type: "boolean"}

	case "AnyType":
		return &ResolvedType{Type: "any"}

	case "NullType":
		return &ResolvedType{Type: "null"}

	case "ArrayType":
		res := &ResolvedType{Type: "array"}
		if t.ItemType != nil {
			if idx, err := t.ItemType.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
				res.Items = r.resolve(&r.Types[idx], depth+1)
			}
		}
		return res

	case "UnionType":
		res := &ResolvedType{Type: "union"}
		for _, elem := range t.Elements {
			if idx, err := elem.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
				res.OneOf = append(res.OneOf, r.resolve(&r.Types[idx], depth+1))
			}
		}
		return res

	case "ObjectType":
		res := &ResolvedType{Type: "object"}
		if t.Name != nil {
			res.Name = *t.Name
		}
		if len(t.Properties) > 0 {
			res.Properties = make(map[string]*ResolvedType, len(t.Properties))
			for name, prop := range t.Properties {
				propRes := &ResolvedType{}
				if idx, err := prop.Type.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
					propRes = r.resolve(&r.Types[idx], depth+1)
				}
				if prop.Description != "" {
					propRes.Description = prop.Description
				}
				if prop.IsRequired() {
					b := true
					propRes.Required = &b
				}
				if prop.IsReadOnly() {
					b := true
					propRes.ReadOnly = &b
				}
				if prop.IsWriteOnly() {
					b := true
					propRes.WriteOnly = &b
				}
				res.Properties[name] = propRes
			}
		}
		return res

	case "DiscriminatedObjectType":
		// Treat as an object: merge baseProperties and present discriminated variants as oneOf.
		res := &ResolvedType{Type: "object"}
		if t.Name != nil {
			res.Name = *t.Name
		}
		allProps := make(map[string]PropertyDef)
		for k, v := range t.BaseProperties {
			allProps[k] = v
		}
		if len(allProps) > 0 {
			res.Properties = make(map[string]*ResolvedType, len(allProps))
			for name, prop := range allProps {
				propRes := &ResolvedType{}
				if idx, err := prop.Type.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
					propRes = r.resolve(&r.Types[idx], depth+1)
				}
				if prop.Description != "" {
					propRes.Description = prop.Description
				}
				if prop.IsRequired() {
					b := true
					propRes.Required = &b
				}
				if prop.IsReadOnly() {
					b := true
					propRes.ReadOnly = &b
				}
				if prop.IsWriteOnly() {
					b := true
					propRes.WriteOnly = &b
				}
				res.Properties[name] = propRes
			}
		}
		keys := make([]string, 0, len(t.ElementMap))
		for k := range t.ElementMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ref := t.ElementMap[k]
			if idx, err := ref.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
				res.OneOf = append(res.OneOf, r.resolve(&r.Types[idx], depth+1))
			}
		}
		return res

	default:
		return &ResolvedType{Type: typeStr(t.Type)}
	}
}

func typeStr(t string) string {
	if t == "" {
		return "unknown"
	}
	return t
}

// TypeString returns a short type string for a TypeEntry (used by the summary renderer).
func (r *Resolver) TypeString(t *TypeEntry) string {
	return r.typeString(t, 0)
}

func (r *Resolver) typeString(t *TypeEntry, depth int) string {
	switch t.Type {
	case "StringType":
		return "string"
	case "StringLiteralType":
		if t.Value != nil {
			return fmt.Sprintf("%q", *t.Value)
		}
		return `""`
	case "IntegerType":
		return "integer"
	case "BooleanType":
		return "boolean"
	case "AnyType":
		return "any"
	case "NullType":
		return "null"
	case "ArrayType":
		if t.ItemType != nil {
			if idx, err := t.ItemType.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
				return "array<" + r.typeString(&r.Types[idx], depth+1) + ">"
			}
		}
		return "array"
	case "UnionType":
		parts := make([]string, 0, len(t.Elements))
		for _, elem := range t.Elements {
			if idx, err := elem.Index(); err == nil && idx >= 0 && idx < len(r.Types) {
				parts = append(parts, r.typeString(&r.Types[idx], depth+1))
			} else {
				parts = append(parts, "?")
			}
		}
		return "(" + joinStrings(parts, " | ") + ")"
	case "ObjectType":
		if t.Name != nil {
			return *t.Name
		}
		return "object"
	case "DiscriminatedObjectType":
		if t.Name != nil {
			return *t.Name
		}
		return "object"
	case "ResourceType":
		if t.Name != nil {
			return *t.Name
		}
		return "resource"
	default:
		if t.Type == "" {
			return "unknown"
		}
		return t.Type
	}
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

// BodyType returns the ObjectType body entry for a ResourceType at the given index.
func (r *Resolver) BodyType(typeIndex int) (*TypeEntry, error) {
	if typeIndex < 0 || typeIndex >= len(r.Types) {
		return nil, fmt.Errorf("type index %d out of range", typeIndex)
	}

	rt := r.Types[typeIndex]
	if rt.Type != "ResourceType" {
		return nil, fmt.Errorf("expected ResourceType at index %d, got %s", typeIndex, rt.Type)
	}

	if rt.Body == nil {
		return nil, fmt.Errorf("ResourceType has no body ref")
	}

	bodyIdx, err := rt.Body.Index()
	if err != nil {
		return nil, fmt.Errorf("invalid body ref: %w", err)
	}

	if bodyIdx < 0 || bodyIdx >= len(r.Types) {
		return nil, fmt.Errorf("body index %d out of range", bodyIdx)
	}

	return &r.Types[bodyIdx], nil
}

// SortedPropertyNames returns property names in alphabetical order.
func SortedPropertyNames(props map[string]PropertyDef) []string {
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
