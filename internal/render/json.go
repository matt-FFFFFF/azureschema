// Package render provides output formatters for Azure resource type schemas.
package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/matt-FFFFFF/azureschema/internal/bicep"
)

// JSON renders the resolved type as formatted JSON to the writer.
func JSON(w io.Writer, resolver *bicep.Resolver, typeIndex int) error {
	resolved, err := resolver.ResolveResourceType(typeIndex)
	if err != nil {
		return fmt.Errorf("resolving resource type: %w", err)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(resolved)
}
