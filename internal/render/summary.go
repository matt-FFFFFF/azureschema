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

	// Collect required top-level properties
	var required []string
	for _, name := range bicep.SortedPropertyNames(body.Properties) {
		prop := body.Properties[name]
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

// printProps recursively prints properties of an ObjectType with indentation.
func printProps(w io.Writer, resolver *bicep.Resolver, t *bicep.TypeEntry, indent string, depth int) {
	if t.Properties == nil {
		return
	}

	for _, name := range bicep.SortedPropertyNames(t.Properties) {
		prop := t.Properties[name]

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

		// Recurse into nested ObjectType properties
		if resolved != nil && resolved.Type == "ObjectType" && resolved.Properties != nil {
			if depth < resolver.MaxDepth {
				printProps(w, resolver, resolved, indent+"    ", depth+1)
			} else {
				fmt.Fprintf(w, "%s      (...depth limit exceeded)\n", indent)
			}
		}
	}
}
