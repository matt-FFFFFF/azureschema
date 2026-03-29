package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/matt-FFFFFF/azureschema/internal/bicep"
)

const (
	headerLine = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	sepLine    = "───────────────────────────────────────────────────────────────────────────────"
	maxDescLen = 120
)

// effectiveProperties returns the merged property map for a TypeEntry,
// combining Properties and BaseProperties for DiscriminatedObjectType.
func effectiveProperties(t *bicep.TypeEntry) map[string]bicep.PropertyDef {
	if t.Type != "DiscriminatedObjectType" || len(t.BaseProperties) == 0 {
		return t.Properties
	}
	merged := make(map[string]bicep.PropertyDef, len(t.Properties)+len(t.BaseProperties))
	for k, v := range t.Properties {
		merged[k] = v
	}
	for k, v := range t.BaseProperties {
		merged[k] = v
	}
	return merged
}

// Summary renders a human-readable summary of the resource type schema.
func Summary(w io.Writer, resolver *bicep.Resolver, typeIndex int, resourceType, apiVersion string) error {
	body, err := resolver.BodyType(typeIndex)
	if err != nil {
		return fmt.Errorf("resolving body type: %w", err)
	}

	// Header
	fmt.Fprintln(w, headerLine)
	fmt.Fprintf(w, "  %s @ %s\n", resourceType, apiVersion)
	fmt.Fprintln(w, headerLine)
	fmt.Fprintln(w)

	// Properties section
	fmt.Fprintln(w, "PROPERTIES:")
	fmt.Fprintln(w, sepLine)

	printProps(w, resolver, body, "", 0)

	// Collect required top-level properties (including baseProperties for DiscriminatedObjectType).
	allProps := effectiveProperties(body)
	var required []string
	for _, name := range bicep.SortedPropertyNames(allProps) {
		prop := allProps[name]
		if prop.IsRequired() {
			required = append(required, name)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, sepLine)
	fmt.Fprintf(w, "Required: %s\n", strings.Join(required, ", "))
	fmt.Fprintln(w)

	return nil
}

// printProps recursively prints properties of an ObjectType or DiscriminatedObjectType with indentation.
func printProps(w io.Writer, resolver *bicep.Resolver, t *bicep.TypeEntry, indent string, depth int) {
	props := effectiveProperties(t)
	if props == nil {
		return
	}

	for _, name := range bicep.SortedPropertyNames(props) {
		prop := props[name]

		// Resolve the type ref to get the actual type entry.
		var resolved *bicep.TypeEntry
		if idx, err := prop.Type.Index(); err == nil && idx >= 0 && idx < len(resolver.Types) {
			resolved = &resolver.Types[idx]
		}

		// Type string
		tstr := "unknown"
		if resolved != nil {
			tstr = resolver.TypeString(resolved)
		}

		// Flags
		var flags string
		if prop.IsRequired() {
			flags += " [REQUIRED]"
		}
		if prop.IsReadOnly() {
			flags += " [READ-ONLY]"
		}
		if prop.IsWriteOnly() {
			flags += " [WRITE-ONLY]"
		}

		// Description
		var desc string
		if prop.Description != "" {
			d := prop.Description
			if len(d) > maxDescLen {
				d = d[:maxDescLen] + "..."
			}
			desc = fmt.Sprintf("\n%s      %s", indent, d)
		}

		fmt.Fprintf(w, "%s  %s: %s%s%s\n", indent, name, tstr, flags, desc)

		// Recurse into nested ObjectType or DiscriminatedObjectType properties
		if resolved != nil && (resolved.Type == "ObjectType" || resolved.Type == "DiscriminatedObjectType") && len(effectiveProperties(resolved)) > 0 {
			if depth < resolver.MaxDepth {
				printProps(w, resolver, resolved, indent+"    ", depth+1)
			} else {
				fmt.Fprintf(w, "%s      (...depth limit exceeded)\n", indent)
			}
		}
	}
}
