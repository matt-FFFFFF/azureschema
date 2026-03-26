package bicep

import (
	"fmt"
	"strconv"
	"strings"
)

// IndexRef holds the resolved file path and type index from the index.json lookup.
type IndexRef struct {
	FilePath  string
	TypeIndex int
}

// LookupResource finds a resource type in the index, returning the file path and type index.
// The lookup is case-insensitive.
func LookupResource(idx *IndexFile, resourceType, apiVersion string) (*IndexRef, error) {
	lookupKey := resourceType + "@" + apiVersion

	// Try exact match first.
	if ref, ok := idx.Resources[lookupKey]; ok {
		return parseRef(ref.Ref)
	}

	// Case-insensitive fallback.
	lowerKey := strings.ToLower(lookupKey)
	for key, ref := range idx.Resources {
		if strings.ToLower(key) == lowerKey {
			return parseRef(ref.Ref)
		}
	}

	return nil, fmt.Errorf("resource type '%s' not found in index", lookupKey)
}

// parseRef parses a ref string like "containerservice_0/microsoft.containerservice/2025-10-01/types.json#/376"
// into a file path and type index.
func parseRef(ref string) (*IndexRef, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty ref")
	}

	parts := strings.SplitN(ref, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ref format (missing #): %s", ref)
	}

	filePath := parts[0]
	indexStr := strings.TrimPrefix(parts[1], "/")

	typeIndex, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid type index in ref %q: %w", ref, err)
	}

	return &IndexRef{
		FilePath:  filePath,
		TypeIndex: typeIndex,
	}, nil
}

// ListVersions returns all resource type/version entries matching a provider or resource type
// prefix (case-insensitive). Each result is a pair of [resourceType, apiVersion].
//
// When the input contains a '/' (i.e. it specifies a resource type, not just a provider),
// only exact resource type matches are returned. When the input is provider-only (no '/'),
// all resource types under that provider are returned.
func ListVersions(idx *IndexFile, provider string) [][2]string {
	lowerPrefix := strings.ToLower(provider)
	// If the input contains '/', the user specified a resource type — match exactly.
	// Otherwise they specified a provider — allow '/' boundaries for sub-types.
	isResourceType := strings.Contains(lowerPrefix, "/")
	var results [][2]string

	for key := range idx.Resources {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, lowerPrefix) {
			// Check what follows the prefix to avoid partial name matches.
			rest := lowerKey[len(lowerPrefix):]
			if len(lowerPrefix) > 0 && len(rest) > 0 {
				if isResourceType {
					// Resource type query: only '@' is valid (exact type match).
					if rest[0] != '@' {
						continue
					}
				} else {
					// Provider query: '@' or '/' are valid boundaries.
					if rest[0] != '@' && rest[0] != '/' {
						continue
					}
				}
			}
			parts := strings.SplitN(key, "@", 2)
			if len(parts) == 2 {
				results = append(results, [2]string{parts[0], parts[1]})
			}
		}
	}

	return results
}
