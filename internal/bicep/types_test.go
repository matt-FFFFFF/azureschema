package bicep

import (
	"encoding/json"
	"testing"
)

// --- Ref.Index tests ---

func TestRefIndex(t *testing.T) {
	tests := []struct {
		name    string
		ref     Ref
		want    int
		wantErr bool
	}{
		{
			name: "standard ref",
			ref:  Ref{Ref: "#/42"},
			want: 42,
		},
		{
			name: "zero index",
			ref:  Ref{Ref: "#/0"},
			want: 0,
		},
		{
			name: "large index",
			ref:  Ref{Ref: "#/376"},
			want: 376,
		},
		{
			name:    "empty ref",
			ref:     Ref{Ref: ""},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "no slash",
			ref:     Ref{Ref: "noslash"},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "non-numeric index",
			ref:     Ref{Ref: "#/abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ref.Index()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// --- PropertyDef flag tests ---

func TestPropertyDefFlags(t *testing.T) {
	tests := []struct {
		name      string
		flags     int
		required  bool
		readOnly  bool
		writeOnly bool
	}{
		{"no flags", 0, false, false, false},
		{"required only", FlagRequired, true, false, false},
		{"read-only only", FlagReadOnly, false, true, false},
		{"write-only only", FlagWriteOnly, false, false, true},
		{"required + read-only", FlagRequired | FlagReadOnly, true, true, false},
		{"all flags", FlagRequired | FlagReadOnly | FlagWriteOnly, true, true, true},
		{"required + deploy-time", FlagRequired | FlagDeployTimeConst, true, false, false},
		// Real-world examples: flags=9 is Required|DeployTimeConst, flags=10 is ReadOnly|DeployTimeConst
		{"flags 9 (req+deploy)", 9, true, false, false},
		{"flags 10 (ro+deploy)", 10, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PropertyDef{Flags: tt.flags}
			if got := p.IsRequired(); got != tt.required {
				t.Errorf("IsRequired() = %v, want %v", got, tt.required)
			}
			if got := p.IsReadOnly(); got != tt.readOnly {
				t.Errorf("IsReadOnly() = %v, want %v", got, tt.readOnly)
			}
			if got := p.IsWriteOnly(); got != tt.writeOnly {
				t.Errorf("IsWriteOnly() = %v, want %v", got, tt.writeOnly)
			}
		})
	}
}

// --- TypeEntry UnmarshalJSON tests ---

func TestTypeEntryUnmarshalJSON(t *testing.T) {
	t.Run("StringType", func(t *testing.T) {
		data := []byte(`{"$type": "StringType"}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "StringType" {
			t.Errorf("Type = %q, want %q", te.Type, "StringType")
		}
	})

	t.Run("StringType with constraints", func(t *testing.T) {
		data := []byte(`{"$type": "StringType", "minLength": 3, "maxLength": 50, "pattern": "^[a-z]+$"}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "StringType" {
			t.Errorf("Type = %q, want %q", te.Type, "StringType")
		}
		if te.MinLength == nil || *te.MinLength != 3 {
			t.Errorf("MinLength = %v, want 3", te.MinLength)
		}
		if te.MaxLength == nil || *te.MaxLength != 50 {
			t.Errorf("MaxLength = %v, want 50", te.MaxLength)
		}
		if te.Pattern == nil || *te.Pattern != "^[a-z]+$" {
			t.Errorf("Pattern = %v, want ^[a-z]+$", te.Pattern)
		}
	})

	t.Run("StringLiteralType", func(t *testing.T) {
		data := []byte(`{"$type": "StringLiteralType", "value": "Essential"}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "StringLiteralType" {
			t.Errorf("Type = %q, want %q", te.Type, "StringLiteralType")
		}
		if te.Value == nil || *te.Value != "Essential" {
			t.Errorf("Value = %v, want Essential", te.Value)
		}
	})

	t.Run("IntegerType with constraints", func(t *testing.T) {
		data := []byte(`{"$type": "IntegerType", "minValue": 0, "maxValue": 100}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "IntegerType" {
			t.Errorf("Type = %q, want %q", te.Type, "IntegerType")
		}
		if te.MinValue == nil || *te.MinValue != 0 {
			t.Errorf("MinValue = %v, want 0", te.MinValue)
		}
		if te.MaxValue == nil || *te.MaxValue != 100 {
			t.Errorf("MaxValue = %v, want 100", te.MaxValue)
		}
	})

	t.Run("BooleanType", func(t *testing.T) {
		data := []byte(`{"$type": "BooleanType"}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "BooleanType" {
			t.Errorf("Type = %q, want %q", te.Type, "BooleanType")
		}
	})

	t.Run("AnyType", func(t *testing.T) {
		data := []byte(`{"$type": "AnyType"}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "AnyType" {
			t.Errorf("Type = %q, want %q", te.Type, "AnyType")
		}
	})

	t.Run("ArrayType", func(t *testing.T) {
		data := []byte(`{"$type": "ArrayType", "itemType": {"$ref": "#/0"}}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "ArrayType" {
			t.Errorf("Type = %q, want %q", te.Type, "ArrayType")
		}
		if te.ItemType == nil || te.ItemType.Ref != "#/0" {
			t.Errorf("ItemType = %v, want ref #/0", te.ItemType)
		}
	})

	t.Run("UnionType", func(t *testing.T) {
		data := []byte(`{"$type": "UnionType", "elements": [{"$ref": "#/1"}, {"$ref": "#/2"}]}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "UnionType" {
			t.Errorf("Type = %q, want %q", te.Type, "UnionType")
		}
		if len(te.Elements) != 2 {
			t.Fatalf("Elements length = %d, want 2", len(te.Elements))
		}
		if te.Elements[0].Ref != "#/1" {
			t.Errorf("Elements[0].Ref = %q, want %q", te.Elements[0].Ref, "#/1")
		}
		if te.Elements[1].Ref != "#/2" {
			t.Errorf("Elements[1].Ref = %q, want %q", te.Elements[1].Ref, "#/2")
		}
	})

	t.Run("ObjectType with properties", func(t *testing.T) {
		data := []byte(`{
			"$type": "ObjectType",
			"name": "TestObject",
			"properties": {
				"id": {"type": {"$ref": "#/0"}, "flags": 10, "description": "The resource id"},
				"name": {"type": {"$ref": "#/0"}, "flags": 1, "description": "The name"}
			}
		}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "ObjectType" {
			t.Errorf("Type = %q, want %q", te.Type, "ObjectType")
		}
		if te.Name == nil || *te.Name != "TestObject" {
			t.Errorf("Name = %v, want TestObject", te.Name)
		}
		if len(te.Properties) != 2 {
			t.Fatalf("Properties length = %d, want 2", len(te.Properties))
		}
		idProp := te.Properties["id"]
		if idProp.Flags != 10 {
			t.Errorf("id.Flags = %d, want 10", idProp.Flags)
		}
		if idProp.Description != "The resource id" {
			t.Errorf("id.Description = %q, want %q", idProp.Description, "The resource id")
		}
		nameProp := te.Properties["name"]
		if !nameProp.IsRequired() {
			t.Error("name should be required")
		}
	})

	t.Run("ResourceType", func(t *testing.T) {
		data := []byte(`{"$type": "ResourceType", "name": "Microsoft.Test/resources@2023-01-01", "body": {"$ref": "#/7"}}`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if te.Type != "ResourceType" {
			t.Errorf("Type = %q, want %q", te.Type, "ResourceType")
		}
		if te.Name == nil || *te.Name != "Microsoft.Test/resources@2023-01-01" {
			t.Errorf("Name = %v, want Microsoft.Test/resources@2023-01-01", te.Name)
		}
		if te.Body == nil || te.Body.Ref != "#/7" {
			t.Errorf("Body = %v, want ref #/7", te.Body)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`{not valid json`)
		var te TypeEntry
		if err := json.Unmarshal(data, &te); err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

// --- ParseTypesFile tests ---

func TestParseTypesFile(t *testing.T) {
	t.Run("valid types file", func(t *testing.T) {
		data := []byte(`[
			{"$type": "StringType"},
			{"$type": "StringLiteralType", "value": "Hello"},
			{"$type": "BooleanType"}
		]`)
		types, err := ParseTypesFile(data)
		if err != nil {
			t.Fatalf("ParseTypesFile: %v", err)
		}
		if len(types) != 3 {
			t.Fatalf("len = %d, want 3", len(types))
		}
		if types[0].Type != "StringType" {
			t.Errorf("[0].Type = %q, want StringType", types[0].Type)
		}
		if types[1].Type != "StringLiteralType" {
			t.Errorf("[1].Type = %q, want StringLiteralType", types[1].Type)
		}
		if types[1].Value == nil || *types[1].Value != "Hello" {
			t.Errorf("[1].Value = %v, want Hello", types[1].Value)
		}
		if types[2].Type != "BooleanType" {
			t.Errorf("[2].Type = %q, want BooleanType", types[2].Type)
		}
	})

	t.Run("empty array", func(t *testing.T) {
		types, err := ParseTypesFile([]byte(`[]`))
		if err != nil {
			t.Fatalf("ParseTypesFile: %v", err)
		}
		if len(types) != 0 {
			t.Errorf("len = %d, want 0", len(types))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseTypesFile([]byte(`not json`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- ParseIndexFile tests ---

func TestParseIndexFile(t *testing.T) {
	t.Run("valid index", func(t *testing.T) {
		data := []byte(`{
			"resources": {
				"Microsoft.Test/resources@2023-01-01": {"$ref": "test/microsoft.test/2023-01-01/types.json#/17"},
				"Microsoft.Test/resources@2024-01-01": {"$ref": "test/microsoft.test/2024-01-01/types.json#/25"}
			}
		}`)
		idx, err := ParseIndexFile(data)
		if err != nil {
			t.Fatalf("ParseIndexFile: %v", err)
		}
		if len(idx.Resources) != 2 {
			t.Fatalf("Resources length = %d, want 2", len(idx.Resources))
		}
		ref, ok := idx.Resources["Microsoft.Test/resources@2023-01-01"]
		if !ok {
			t.Fatal("missing Microsoft.Test/resources@2023-01-01")
		}
		if ref.Ref != "test/microsoft.test/2023-01-01/types.json#/17" {
			t.Errorf("ref = %q", ref.Ref)
		}
	})

	t.Run("empty resources", func(t *testing.T) {
		idx, err := ParseIndexFile([]byte(`{"resources": {}}`))
		if err != nil {
			t.Fatalf("ParseIndexFile: %v", err)
		}
		if len(idx.Resources) != 0 {
			t.Errorf("Resources length = %d, want 0", len(idx.Resources))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseIndexFile([]byte(`not json`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- Full types.json round-trip (matches real bicep-types-az format) ---

func TestParseRealWorldTypesJSON(t *testing.T) {
	// Minimal realistic types.json based on Microsoft.Addons
	data := []byte(`[
		{"$type":"StringType"},
		{"$type":"StringLiteralType","value":"Essential"},
		{"$type":"StringLiteralType","value":"Standard"},
		{"$type":"StringLiteralType","value":"Advanced"},
		{"$type":"UnionType","elements":[{"$ref":"#/1"},{"$ref":"#/2"},{"$ref":"#/3"},{"$ref":"#/0"}]},
		{"$type":"StringLiteralType","value":"Microsoft.Addons/supportProviders/supportPlanTypes"},
		{"$type":"StringLiteralType","value":"2017-05-15"},
		{"$type":"ObjectType","name":"Microsoft.Addons/supportProviders/supportPlanTypes","properties":{"id":{"type":{"$ref":"#/0"},"flags":10,"description":"The resource id"},"name":{"type":{"$ref":"#/4"},"flags":9,"description":"The resource name"},"type":{"type":{"$ref":"#/5"},"flags":10,"description":"The resource type"},"apiVersion":{"type":{"$ref":"#/6"},"flags":10,"description":"The resource api version"}}},
		{"$type":"ResourceType","name":"Microsoft.Addons/supportProviders/supportPlanTypes@2017-05-15","body":{"$ref":"#/7"}}
	]`)

	types, err := ParseTypesFile(data)
	if err != nil {
		t.Fatalf("ParseTypesFile: %v", err)
	}
	if len(types) != 9 {
		t.Fatalf("len = %d, want 9", len(types))
	}

	// Verify the ResourceType at index 8
	rt := types[8]
	if rt.Type != "ResourceType" {
		t.Errorf("types[8].Type = %q, want ResourceType", rt.Type)
	}
	if rt.Body == nil {
		t.Fatal("types[8].Body is nil")
	}
	idx, err := rt.Body.Index()
	if err != nil {
		t.Fatalf("Body.Index: %v", err)
	}
	if idx != 7 {
		t.Errorf("Body index = %d, want 7", idx)
	}

	// Verify the body ObjectType at index 7
	body := types[7]
	if body.Type != "ObjectType" {
		t.Errorf("types[7].Type = %q, want ObjectType", body.Type)
	}
	if len(body.Properties) != 4 {
		t.Errorf("properties count = %d, want 4", len(body.Properties))
	}

	// Check name property is required (flags=9 = Required|DeployTimeConst)
	nameProp := body.Properties["name"]
	if !nameProp.IsRequired() {
		t.Error("name should be required")
	}

	// Check id property is read-only (flags=10 = ReadOnly|DeployTimeConst)
	idProp := body.Properties["id"]
	if !idProp.IsReadOnly() {
		t.Error("id should be read-only")
	}
	if idProp.IsRequired() {
		t.Error("id should not be required")
	}
}
