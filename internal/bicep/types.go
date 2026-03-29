// Package bicep provides types and utilities for working with Azure bicep-types-az data.
package bicep

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Bitfield flags for property attributes.
const (
	FlagRequired        = 1
	FlagReadOnly        = 2
	FlagWriteOnly       = 4
	FlagDeployTimeConst = 8
)

// TypeEntry is a discriminated union for all bicep type entries.
// The concrete type is determined by the Type field ("$type" in JSON).
type TypeEntry struct {
	Type string // "$type" discriminator

	// StringType fields
	MinLength *int    `json:"minLength,omitempty"`
	MaxLength *int    `json:"maxLength,omitempty"`
	Pattern   *string `json:"pattern,omitempty"`

	// StringLiteralType fields
	Value *string `json:"value,omitempty"`

	// IntegerType fields
	MinValue *int64 `json:"minValue,omitempty"`
	MaxValue *int64 `json:"maxValue,omitempty"`

	// ObjectType / DiscriminatedObjectType fields
	Name           *string                `json:"name,omitempty"`
	Properties     map[string]PropertyDef `json:"properties,omitempty"`
	BaseProperties map[string]PropertyDef `json:"baseProperties,omitempty"`
	Discriminator  *string                `json:"discriminator,omitempty"`

	// ArrayType fields
	ItemType *Ref `json:"itemType,omitempty"`

	// UnionType fields — array of $ref
	Elements []Ref `json:"elements,omitempty"`

	// DiscriminatedObjectType fields — map of name → $ref
	ElementMap map[string]Ref `json:"-"`

	// ResourceType fields
	Body *Ref `json:"body,omitempty"`

	// ResourceFunctionType fields
	ResourceType *string `json:"resourceType,omitempty"`
	ApiVersion   *string `json:"apiVersion,omitempty"`
	Output       *Ref    `json:"output,omitempty"`
	Input        *Ref    `json:"input,omitempty"`
}

// PropertyDef describes a property within an ObjectType.
type PropertyDef struct {
	Type        Ref    `json:"type"`
	Flags       int    `json:"flags"`
	Description string `json:"description,omitempty"`
}

// Ref is a JSON "$ref" pointer, e.g. "#/42".
type Ref struct {
	Ref string `json:"$ref,omitempty"`
}

// Index returns the integer type index from a ref like "#/42".
func (r Ref) Index() (int, error) {
	if r.Ref == "" {
		return -1, fmt.Errorf("empty ref")
	}
	parts := strings.Split(r.Ref, "/")
	if len(parts) < 2 {
		return -1, fmt.Errorf("invalid ref format: %s", r.Ref)
	}
	return strconv.Atoi(parts[len(parts)-1])
}

// IsRequired returns true if the Required bit is set.
func (p PropertyDef) IsRequired() bool { return p.Flags&FlagRequired != 0 }

// IsReadOnly returns true if the ReadOnly bit is set.
func (p PropertyDef) IsReadOnly() bool { return p.Flags&FlagReadOnly != 0 }

// IsWriteOnly returns true if the WriteOnly bit is set.
func (p PropertyDef) IsWriteOnly() bool { return p.Flags&FlagWriteOnly != 0 }

// UnmarshalJSON implements custom unmarshalling to handle the "$type" discriminator
// and the polymorphic "elements" field ([]Ref for UnionType, map[string]Ref for DiscriminatedObjectType).
func (t *TypeEntry) UnmarshalJSON(data []byte) error {
	// First extract the discriminator.
	var disc struct {
		Type string `json:"$type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	t.Type = disc.Type

	// Use an alias to avoid infinite recursion, but omit the "elements" field
	// since it is polymorphic: []Ref for UnionType, map[string]Ref for DiscriminatedObjectType.
	type Alias TypeEntry
	aux := &struct {
		*Alias
		// Shadow both elements fields so the alias does not try to decode them.
		Elements   json.RawMessage `json:"elements"`
		ElementMap json.RawMessage `json:"-"`
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Now decode "elements" according to the concrete type.
	if len(aux.Elements) > 0 {
		if string(aux.Elements) == "null" {
			// Match encoding/json semantics: a present null clears the field.
			t.Elements = nil
			t.ElementMap = nil
		} else {
			switch t.Type {
			case "DiscriminatedObjectType":
				if err := json.Unmarshal(aux.Elements, &t.ElementMap); err != nil {
					return fmt.Errorf("parsing DiscriminatedObjectType elements: %w", err)
				}
			default:
				if err := json.Unmarshal(aux.Elements, &t.Elements); err != nil {
					return fmt.Errorf("parsing elements: %w", err)
				}
			}
		}
	}
	return nil
}

// TypesFile is a parsed types.json - an array of TypeEntry.
type TypesFile []TypeEntry

// ParseTypesFile parses a types.json byte slice.
func ParseTypesFile(data []byte) (TypesFile, error) {
	var types TypesFile
	if err := json.Unmarshal(data, &types); err != nil {
		return nil, fmt.Errorf("parsing types file: %w", err)
	}
	return types, nil
}

// IndexFile represents the top-level index.json structure.
type IndexFile struct {
	Resources map[string]Ref `json:"resources"`
}

// ParseIndexFile parses an index.json byte slice.
func ParseIndexFile(data []byte) (*IndexFile, error) {
	var idx IndexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index file: %w", err)
	}
	return &idx, nil
}
