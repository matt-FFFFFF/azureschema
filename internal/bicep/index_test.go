package bicep

import (
	"sort"
	"testing"
)

func testIndex() *IndexFile {
	return &IndexFile{
		Resources: map[string]Ref{
			"Microsoft.Storage/storageAccounts@2023-01-01":                                    {Ref: "storage/microsoft.storage/2023-01-01/types.json#/42"},
			"Microsoft.Storage/storageAccounts@2024-01-01":                                    {Ref: "storage/microsoft.storage/2024-01-01/types.json#/55"},
			"Microsoft.Storage/storageAccounts/blobServices@2023-01-01":                       {Ref: "storage/microsoft.storage/2023-01-01/types.json#/100"},
			"Microsoft.ContainerService/managedClusters@2025-10-01":                           {Ref: "containerservice_0/microsoft.containerservice/2025-10-01/types.json#/376"},
			"Microsoft.ContainerService/managedClustersSnapshots@2025-10-01":                  {Ref: "containerservice_0/microsoft.containerservice/2025-10-01/types.json#/400"},
			"Microsoft.ContainerService/managedClusters/maintenanceConfigurations@2025-10-01": {Ref: "containerservice_0/microsoft.containerservice/2025-10-01/types.json#/410"},
			"Microsoft.Addons/supportProviders/supportPlanTypes@2017-05-15":                   {Ref: "addons/microsoft.addons/2017-05-15/types.json#/17"},
		},
	}
}

// --- LookupResource tests ---

func TestLookupResource(t *testing.T) {
	idx := testIndex()

	t.Run("exact match", func(t *testing.T) {
		ref, err := LookupResource(idx, "Microsoft.Storage/storageAccounts", "2023-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.FilePath != "storage/microsoft.storage/2023-01-01/types.json" {
			t.Errorf("FilePath = %q", ref.FilePath)
		}
		if ref.TypeIndex != 42 {
			t.Errorf("TypeIndex = %d, want 42", ref.TypeIndex)
		}
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		ref, err := LookupResource(idx, "microsoft.storage/storageaccounts", "2023-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.FilePath != "storage/microsoft.storage/2023-01-01/types.json" {
			t.Errorf("FilePath = %q", ref.FilePath)
		}
		if ref.TypeIndex != 42 {
			t.Errorf("TypeIndex = %d, want 42", ref.TypeIndex)
		}
	})

	t.Run("different api version", func(t *testing.T) {
		ref, err := LookupResource(idx, "Microsoft.Storage/storageAccounts", "2024-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.TypeIndex != 55 {
			t.Errorf("TypeIndex = %d, want 55", ref.TypeIndex)
		}
	})

	t.Run("not found resource type", func(t *testing.T) {
		_, err := LookupResource(idx, "Microsoft.Fake/nonexistent", "2023-01-01")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not found api version", func(t *testing.T) {
		_, err := LookupResource(idx, "Microsoft.Storage/storageAccounts", "1999-01-01")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nested resource type", func(t *testing.T) {
		ref, err := LookupResource(idx, "Microsoft.Storage/storageAccounts/blobServices", "2023-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.TypeIndex != 100 {
			t.Errorf("TypeIndex = %d, want 100", ref.TypeIndex)
		}
	})

	t.Run("mixed case exact", func(t *testing.T) {
		ref, err := LookupResource(idx, "MICROSOFT.STORAGE/STORAGEACCOUNTS", "2023-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.TypeIndex != 42 {
			t.Errorf("TypeIndex = %d, want 42", ref.TypeIndex)
		}
	})
}

// --- parseRef tests ---

func TestParseRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantPath  string
		wantIndex int
		wantErr   bool
	}{
		{
			name:      "standard ref",
			ref:       "storage/microsoft.storage/2023-01-01/types.json#/42",
			wantPath:  "storage/microsoft.storage/2023-01-01/types.json",
			wantIndex: 42,
		},
		{
			name:      "zero index",
			ref:       "test/types.json#/0",
			wantPath:  "test/types.json",
			wantIndex: 0,
		},
		{
			name:      "large index",
			ref:       "containerservice_0/microsoft.containerservice/2025-10-01/types.json#/376",
			wantPath:  "containerservice_0/microsoft.containerservice/2025-10-01/types.json",
			wantIndex: 376,
		},
		{
			name:    "empty ref",
			ref:     "",
			wantErr: true,
		},
		{
			name:    "no hash separator",
			ref:     "types.json/42",
			wantErr: true,
		},
		{
			name:    "non-numeric index",
			ref:     "types.json#/abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.FilePath != tt.wantPath {
				t.Errorf("FilePath = %q, want %q", got.FilePath, tt.wantPath)
			}
			if got.TypeIndex != tt.wantIndex {
				t.Errorf("TypeIndex = %d, want %d", got.TypeIndex, tt.wantIndex)
			}
		})
	}
}

// --- ListVersions tests ---

func TestListVersions(t *testing.T) {
	idx := testIndex()

	t.Run("storage provider", func(t *testing.T) {
		results := ListVersions(idx, "Microsoft.Storage")
		if len(results) != 3 {
			t.Fatalf("len = %d, want 3", len(results))
		}
		// Sort for deterministic comparison
		sort.Slice(results, func(i, j int) bool {
			if results[i][0] == results[j][0] {
				return results[i][1] < results[j][1]
			}
			return results[i][0] < results[j][0]
		})
		if results[0][0] != "Microsoft.Storage/storageAccounts" || results[0][1] != "2023-01-01" {
			t.Errorf("results[0] = %v", results[0])
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		results := ListVersions(idx, "microsoft.storage")
		if len(results) != 3 {
			t.Fatalf("len = %d, want 3", len(results))
		}
	})

	t.Run("container service", func(t *testing.T) {
		results := ListVersions(idx, "Microsoft.ContainerService")
		if len(results) != 3 {
			t.Fatalf("len = %d, want 3", len(results))
		}
	})

	t.Run("exact resource type excludes prefix matches", func(t *testing.T) {
		// "managedClusters" must NOT match "managedClustersSnapshots"
		results := ListVersions(idx, "Microsoft.ContainerService/managedClusters")
		if len(results) != 1 {
			t.Fatalf("len = %d, want 1", len(results))
		}
		if results[0][0] != "Microsoft.ContainerService/managedClusters" {
			t.Errorf("resource type = %q", results[0][0])
		}
		if results[0][1] != "2025-10-01" {
			t.Errorf("api version = %q", results[0][1])
		}
	})

	t.Run("exact resource type excludes sub-resources", func(t *testing.T) {
		// "managedClusters" must NOT match "managedClusters/maintenanceConfigurations"
		results := ListVersions(idx, "Microsoft.ContainerService/managedClusters")
		if len(results) != 1 {
			t.Fatalf("len = %d, want 1; got %v", len(results), results)
		}
		if results[0][0] != "Microsoft.ContainerService/managedClusters" {
			t.Errorf("resource type = %q, want Microsoft.ContainerService/managedClusters", results[0][0])
		}
	})

	t.Run("no match", func(t *testing.T) {
		results := ListVersions(idx, "Microsoft.Fake")
		if len(results) != 0 {
			t.Errorf("len = %d, want 0", len(results))
		}
	})

	t.Run("empty provider", func(t *testing.T) {
		// Empty string prefix matches everything
		results := ListVersions(idx, "")
		if len(results) != 7 {
			t.Errorf("len = %d, want 7", len(results))
		}
	})
}
