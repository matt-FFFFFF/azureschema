package bicep

import (
	"testing"
)

// buildTestTypes creates a minimal but representative set of type entries
// for testing the resolver. The layout mirrors a simplified real types.json:
//
//	[0] StringType
//	[1] StringLiteralType "Essential"
//	[2] StringLiteralType "Standard"
//	[3] UnionType [#/1, #/2, #/0]
//	[4] IntegerType (minValue=0, maxValue=100)
//	[5] BooleanType
//	[6] AnyType
//	[7] ArrayType (itemType -> #/0)
//	[8] ObjectType "InnerProps" {enabled: #/5 (flags=1,required), count: #/4 (flags=0)}
//	[9] ObjectType "TestResource" {id: #/0 (flags=10,ro), name: #/3 (flags=9,req), props: #/8 (flags=0), tags: #/7 (flags=0)}
//	[10] ResourceType body -> #/9
//	[11] ObjectType "DeepNested" {value: #/0}
//	[12] ObjectType "MidLevel" {nested: #/11}
//	[13] ObjectType "TopLevel" {mid: #/12}
//	[14] ResourceType body -> #/13
//	[15] StringLiteralType "ftp"
//	[16] ObjectType "FtpPolicy" {name: #/15 (flags=1,required)}
//	[17] StringLiteralType "scm"
//	[18] ObjectType "ScmPolicy" {name: #/17 (flags=1,required)}
//	[19] DiscriminatedObjectType "PublishingPolicies" discriminator=name, baseProperties={id: #/0 (flags=2)}, elements={ftp: #/16, scm: #/18}
//	[20] ResourceType body -> #/19
func buildTestTypes() TypesFile {
	strPtr := func(s string) *string { return &s }
	int64Ptr := func(i int64) *int64 { return &i }

	return TypesFile{
		// [0] StringType
		{Type: "StringType"},
		// [1] StringLiteralType "Essential"
		{Type: "StringLiteralType", Value: strPtr("Essential")},
		// [2] StringLiteralType "Standard"
		{Type: "StringLiteralType", Value: strPtr("Standard")},
		// [3] UnionType
		{Type: "UnionType", Elements: []Ref{{Ref: "#/1"}, {Ref: "#/2"}, {Ref: "#/0"}}},
		// [4] IntegerType
		{Type: "IntegerType", MinValue: int64Ptr(0), MaxValue: int64Ptr(100)},
		// [5] BooleanType
		{Type: "BooleanType"},
		// [6] AnyType
		{Type: "AnyType"},
		// [7] ArrayType -> StringType
		{Type: "ArrayType", ItemType: &Ref{Ref: "#/0"}},
		// [8] ObjectType "InnerProps"
		{
			Type: "ObjectType",
			Name: strPtr("InnerProps"),
			Properties: map[string]PropertyDef{
				"enabled": {Type: Ref{Ref: "#/5"}, Flags: FlagRequired, Description: "Enable the feature"},
				"count":   {Type: Ref{Ref: "#/4"}, Flags: 0, Description: "Item count"},
			},
		},
		// [9] ObjectType "TestResource"
		{
			Type: "ObjectType",
			Name: strPtr("TestResource"),
			Properties: map[string]PropertyDef{
				"id":    {Type: Ref{Ref: "#/0"}, Flags: FlagReadOnly | FlagDeployTimeConst, Description: "The resource id"},
				"name":  {Type: Ref{Ref: "#/3"}, Flags: FlagRequired | FlagDeployTimeConst, Description: "The resource name"},
				"props": {Type: Ref{Ref: "#/8"}, Flags: 0, Description: "Properties"},
				"tags":  {Type: Ref{Ref: "#/7"}, Flags: 0, Description: "Resource tags"},
			},
		},
		// [10] ResourceType -> body #/9
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Test/resources@2023-01-01"),
			Body: &Ref{Ref: "#/9"},
		},
		// [11] ObjectType "DeepNested"
		{
			Type: "ObjectType",
			Name: strPtr("DeepNested"),
			Properties: map[string]PropertyDef{
				"value": {Type: Ref{Ref: "#/0"}, Flags: 0, Description: "A value"},
			},
		},
		// [12] ObjectType "MidLevel"
		{
			Type: "ObjectType",
			Name: strPtr("MidLevel"),
			Properties: map[string]PropertyDef{
				"nested": {Type: Ref{Ref: "#/11"}, Flags: 0, Description: "Nested object"},
			},
		},
		// [13] ObjectType "TopLevel"
		{
			Type: "ObjectType",
			Name: strPtr("TopLevel"),
			Properties: map[string]PropertyDef{
				"mid": {Type: Ref{Ref: "#/12"}, Flags: 0, Description: "Mid-level object"},
			},
		},
		// [14] ResourceType -> body #/13
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Test/deep@2023-01-01"),
			Body: &Ref{Ref: "#/13"},
		},
		// [15] StringLiteralType "ftp"
		{Type: "StringLiteralType", Value: strPtr("ftp")},
		// [16] ObjectType "FtpPolicy" {name: #/15 (required)}
		{
			Type: "ObjectType",
			Name: strPtr("FtpPolicy"),
			Properties: map[string]PropertyDef{
				"name": {Type: Ref{Ref: "#/15"}, Flags: FlagRequired, Description: "Policy name"},
			},
		},
		// [17] StringLiteralType "scm"
		{Type: "StringLiteralType", Value: strPtr("scm")},
		// [18] ObjectType "ScmPolicy" {name: #/17 (required)}
		{
			Type: "ObjectType",
			Name: strPtr("ScmPolicy"),
			Properties: map[string]PropertyDef{
				"name": {Type: Ref{Ref: "#/17"}, Flags: FlagRequired, Description: "Policy name"},
			},
		},
		// [19] DiscriminatedObjectType "PublishingPolicies"
		{
			Type:          "DiscriminatedObjectType",
			Name:          strPtr("PublishingPolicies"),
			Discriminator: strPtr("name"),
			BaseProperties: map[string]PropertyDef{
				"id": {Type: Ref{Ref: "#/0"}, Flags: FlagReadOnly, Description: "The resource id"},
			},
			ElementMap: map[string]Ref{
				"ftp": {Ref: "#/16"},
				"scm": {Ref: "#/18"},
			},
		},
		// [20] ResourceType -> body #/19
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Test/discriminated@2023-01-01"),
			Body: &Ref{Ref: "#/19"},
		},
	}
}

// --- ResolveResourceType tests ---

func TestResolveResourceType(t *testing.T) {
	types := buildTestTypes()

	t.Run("basic resolution", func(t *testing.T) {
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolved.Type != "object" {
			t.Errorf("Type = %q, want object", resolved.Type)
		}
		if resolved.Name != "TestResource" {
			t.Errorf("Name = %q, want TestResource", resolved.Name)
		}
		if len(resolved.Properties) != 4 {
			t.Fatalf("Properties count = %d, want 4", len(resolved.Properties))
		}

		// Check id property
		id := resolved.Properties["id"]
		if id.Type != "string" {
			t.Errorf("id.Type = %q, want string", id.Type)
		}
		if id.ReadOnly == nil || !*id.ReadOnly {
			t.Error("id should be readOnly")
		}
		if id.Required != nil {
			t.Error("id should not be required")
		}

		// Check name property (union type)
		name := resolved.Properties["name"]
		if name.Type != "union" {
			t.Errorf("name.Type = %q, want union", name.Type)
		}
		if name.Required == nil || !*name.Required {
			t.Error("name should be required")
		}
		if len(name.OneOf) != 3 {
			t.Fatalf("name.OneOf length = %d, want 3", len(name.OneOf))
		}
		if name.OneOf[0].Const != "Essential" {
			t.Errorf("name.OneOf[0].Const = %q, want Essential", name.OneOf[0].Const)
		}

		// Check props property (nested object)
		props := resolved.Properties["props"]
		if props.Type != "object" {
			t.Errorf("props.Type = %q, want object", props.Type)
		}
		if props.Name != "InnerProps" {
			t.Errorf("props.Name = %q, want InnerProps", props.Name)
		}
		if len(props.Properties) != 2 {
			t.Fatalf("props.Properties count = %d, want 2", len(props.Properties))
		}
		enabled := props.Properties["enabled"]
		if enabled.Type != "boolean" {
			t.Errorf("enabled.Type = %q, want boolean", enabled.Type)
		}
		if enabled.Required == nil || !*enabled.Required {
			t.Error("enabled should be required")
		}

		count := props.Properties["count"]
		if count.Type != "integer" {
			t.Errorf("count.Type = %q, want integer", count.Type)
		}
		if count.Minimum == nil || *count.Minimum != 0 {
			t.Errorf("count.Minimum = %v, want 0", count.Minimum)
		}
		if count.Maximum == nil || *count.Maximum != 100 {
			t.Errorf("count.Maximum = %v, want 100", count.Maximum)
		}

		// Check tags property (array)
		tags := resolved.Properties["tags"]
		if tags.Type != "array" {
			t.Errorf("tags.Type = %q, want array", tags.Type)
		}
		if tags.Items == nil || tags.Items.Type != "string" {
			t.Errorf("tags.Items = %v, want string item", tags.Items)
		}
	})

	t.Run("out of range index", func(t *testing.T) {
		r := NewResolver(types, 5)
		_, err := r.ResolveResourceType(999)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("negative index", func(t *testing.T) {
		r := NewResolver(types, 5)
		_, err := r.ResolveResourceType(-1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not a ResourceType", func(t *testing.T) {
		r := NewResolver(types, 5)
		_, err := r.ResolveResourceType(0) // StringType
		if err == nil {
			t.Fatal("expected error for non-ResourceType")
		}
	})

	t.Run("description preserved", func(t *testing.T) {
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		id := resolved.Properties["id"]
		if id.Description != "The resource id" {
			t.Errorf("id.Description = %q, want %q", id.Description, "The resource id")
		}
	})
}

// --- Depth limiting tests ---

func TestResolverDepthLimit(t *testing.T) {
	types := buildTestTypes()

	t.Run("depth 0 truncates nested objects", func(t *testing.T) {
		r := NewResolver(types, 0)
		resolved, err := r.ResolveResourceType(14) // Deep resource
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// At depth 0, the top-level object is resolved.
		if resolved.Type != "object" {
			t.Errorf("Type = %q, want object", resolved.Type)
		}
		// mid property should resolve at depth 1, which exceeds maxDepth=0
		mid := resolved.Properties["mid"]
		if mid == nil {
			t.Fatal("mid property missing")
		}
		if mid.Truncated != "depth limit exceeded" {
			t.Errorf("mid.Truncated = %q, want 'depth limit exceeded'", mid.Truncated)
		}
	})

	t.Run("depth 1 resolves one level", func(t *testing.T) {
		r := NewResolver(types, 1)
		resolved, err := r.ResolveResourceType(14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mid := resolved.Properties["mid"]
		if mid.Type != "object" {
			t.Errorf("mid.Type = %q, want object", mid.Type)
		}
		if mid.Truncated != "" {
			t.Errorf("mid should not be truncated at depth 1")
		}
		// mid.nested should be truncated at depth 2
		nested := mid.Properties["nested"]
		if nested == nil {
			t.Fatal("nested property missing")
		}
		if nested.Truncated != "depth limit exceeded" {
			t.Errorf("nested.Truncated = %q, want 'depth limit exceeded'", nested.Truncated)
		}
	})

	t.Run("depth 5 resolves everything", func(t *testing.T) {
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mid := resolved.Properties["mid"]
		nested := mid.Properties["nested"]
		if nested.Truncated != "" {
			t.Error("nested should not be truncated at depth 5")
		}
		value := nested.Properties["value"]
		if value == nil {
			t.Fatal("value property missing")
		}
		if value.Type != "string" {
			t.Errorf("value.Type = %q, want string", value.Type)
		}
	})
}

// --- TypeString tests ---

func TestResolverTypeString(t *testing.T) {
	types := buildTestTypes()
	r := NewResolver(types, 5)

	tests := []struct {
		name  string
		index int
		want  string
	}{
		{"StringType", 0, "string"},
		{"StringLiteralType", 1, `"Essential"`},
		{"UnionType", 3, `("Essential" | "Standard" | string)`},
		{"IntegerType", 4, "integer"},
		{"BooleanType", 5, "boolean"},
		{"AnyType", 6, "any"},
		{"ArrayType", 7, "array<string>"},
		{"ObjectType named", 8, "InnerProps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.TypeString(&types[tt.index])
			if got != tt.want {
				t.Errorf("TypeString = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("ObjectType without name", func(t *testing.T) {
		unnamed := TypeEntry{Type: "ObjectType"}
		got := r.TypeString(&unnamed)
		if got != "object" {
			t.Errorf("TypeString = %q, want object", got)
		}
	})

	t.Run("ResourceType", func(t *testing.T) {
		got := r.TypeString(&types[10])
		if got != "Microsoft.Test/resources@2023-01-01" {
			t.Errorf("TypeString = %q", got)
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		unknown := TypeEntry{Type: "SomeNewType"}
		got := r.TypeString(&unknown)
		if got != "SomeNewType" {
			t.Errorf("TypeString = %q, want SomeNewType", got)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		empty := TypeEntry{}
		got := r.TypeString(&empty)
		if got != "unknown" {
			t.Errorf("TypeString = %q, want unknown", got)
		}
	})

	t.Run("NullType", func(t *testing.T) {
		nt := TypeEntry{Type: "NullType"}
		got := r.TypeString(&nt)
		if got != "null" {
			t.Errorf("TypeString = %q, want null", got)
		}
	})

	t.Run("StringLiteralType nil value", func(t *testing.T) {
		slt := TypeEntry{Type: "StringLiteralType"}
		got := r.TypeString(&slt)
		if got != `""` {
			t.Errorf("TypeString = %q, want empty quoted string", got)
		}
	})

	t.Run("ArrayType without itemType", func(t *testing.T) {
		at := TypeEntry{Type: "ArrayType"}
		got := r.TypeString(&at)
		if got != "array" {
			t.Errorf("TypeString = %q, want array", got)
		}
	})
}

// --- BodyType tests ---

func TestResolverBodyType(t *testing.T) {
	types := buildTestTypes()
	r := NewResolver(types, 5)

	t.Run("valid resource type", func(t *testing.T) {
		body, err := r.BodyType(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Type != "ObjectType" {
			t.Errorf("Type = %q, want ObjectType", body.Type)
		}
		if body.Name == nil || *body.Name != "TestResource" {
			t.Errorf("Name = %v, want TestResource", body.Name)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		_, err := r.BodyType(999)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not ResourceType", func(t *testing.T) {
		_, err := r.BodyType(0)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ResourceType without body", func(t *testing.T) {
		typesNoBody := TypesFile{
			{Type: "ResourceType", Name: ptrStr("NoBody")},
		}
		r2 := NewResolver(typesNoBody, 5)
		_, err := r2.BodyType(0)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- SortedPropertyNames tests ---

func TestSortedPropertyNames(t *testing.T) {
	t.Run("sorted output", func(t *testing.T) {
		props := map[string]PropertyDef{
			"zebra":  {},
			"alpha":  {},
			"middle": {},
			"bravo":  {},
		}
		got := SortedPropertyNames(props)
		want := []string{"alpha", "bravo", "middle", "zebra"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d", len(got), len(want))
		}
		for i, g := range got {
			if g != want[i] {
				t.Errorf("[%d] = %q, want %q", i, g, want[i])
			}
		}
	})

	t.Run("nil map", func(t *testing.T) {
		got := SortedPropertyNames(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("empty map", func(t *testing.T) {
		got := SortedPropertyNames(map[string]PropertyDef{})
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

// --- Resolve edge cases ---

func TestResolveEdgeCases(t *testing.T) {
	t.Run("AnyType resolves", func(t *testing.T) {
		types := TypesFile{
			{Type: "AnyType"},
			{
				Type: "ObjectType",
				Name: ptrStr("Obj"),
				Properties: map[string]PropertyDef{
					"data": {Type: Ref{Ref: "#/0"}, Flags: 0},
				},
			},
			{Type: "ResourceType", Body: &Ref{Ref: "#/1"}},
		}
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := resolved.Properties["data"]
		if data.Type != "any" {
			t.Errorf("data.Type = %q, want any", data.Type)
		}
	})

	t.Run("WriteOnly flag", func(t *testing.T) {
		types := TypesFile{
			{Type: "StringType"},
			{
				Type: "ObjectType",
				Name: ptrStr("Obj"),
				Properties: map[string]PropertyDef{
					"secret": {Type: Ref{Ref: "#/0"}, Flags: FlagWriteOnly},
				},
			},
			{Type: "ResourceType", Body: &Ref{Ref: "#/1"}},
		}
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		secret := resolved.Properties["secret"]
		if secret.WriteOnly == nil || !*secret.WriteOnly {
			t.Error("secret should be writeOnly")
		}
	})

	t.Run("NullType resolves", func(t *testing.T) {
		types := TypesFile{
			{Type: "NullType"},
			{
				Type: "ObjectType",
				Name: ptrStr("Obj"),
				Properties: map[string]PropertyDef{
					"nothing": {Type: Ref{Ref: "#/0"}, Flags: 0},
				},
			},
			{Type: "ResourceType", Body: &Ref{Ref: "#/1"}},
		}
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nothing := resolved.Properties["nothing"]
		if nothing.Type != "null" {
			t.Errorf("nothing.Type = %q, want null", nothing.Type)
		}
	})

	t.Run("StringType with constraints", func(t *testing.T) {
		minLen := 3
		maxLen := 50
		pattern := "^[a-z]+$"
		types := TypesFile{
			{Type: "StringType", MinLength: &minLen, MaxLength: &maxLen, Pattern: &pattern},
			{
				Type: "ObjectType",
				Name: ptrStr("Obj"),
				Properties: map[string]PropertyDef{
					"constrained": {Type: Ref{Ref: "#/0"}, Flags: 0},
				},
			},
			{Type: "ResourceType", Body: &Ref{Ref: "#/1"}},
		}
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		c := resolved.Properties["constrained"]
		if c.MinLength == nil || *c.MinLength != 3 {
			t.Errorf("MinLength = %v, want 3", c.MinLength)
		}
		if c.MaxLength == nil || *c.MaxLength != 50 {
			t.Errorf("MaxLength = %v, want 50", c.MaxLength)
		}
		if c.Pattern == nil || *c.Pattern != "^[a-z]+$" {
			t.Errorf("Pattern = %v", c.Pattern)
		}
	})

	t.Run("empty ObjectType (no properties)", func(t *testing.T) {
		types := TypesFile{
			{Type: "ObjectType", Name: ptrStr("Empty")},
			{Type: "ResourceType", Body: &Ref{Ref: "#/0"}},
		}
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resolved.Type != "object" {
			t.Errorf("Type = %q, want object", resolved.Type)
		}
		if len(resolved.Properties) != 0 {
			t.Errorf("Properties count = %d, want 0", len(resolved.Properties))
		}
	})
}

func ptrStr(s string) *string { return &s }

// --- DiscriminatedObjectType depth limit and stable ordering tests ---

func TestDiscriminatedObjectTypeDepthAndOrder(t *testing.T) {
	types := buildTestTypes()

	t.Run("depth limit truncates DiscriminatedObjectType with object type and name", func(t *testing.T) {
		r := NewResolver(types, 0)
		// Resolve from index 20 with depth 0 — body is at index 19 which resolves at depth 0.
		// The id property ref (index 0) will be resolved at depth 1, exceeding maxDepth=0.
		resolved, err := r.ResolveResourceType(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Top-level DiscriminatedObjectType resolves at depth 0 — not truncated.
		if resolved.Type != "object" {
			t.Errorf("Type = %q, want object", resolved.Type)
		}
		if resolved.Name != "PublishingPolicies" {
			t.Errorf("Name = %q, want PublishingPolicies", resolved.Name)
		}
		// id property is resolved at depth 1 which exceeds maxDepth=0, so it should be truncated.
		id := resolved.Properties["id"]
		if id == nil {
			t.Fatal("id property missing")
		}
		if id.Truncated != "depth limit exceeded" {
			t.Errorf("id.Truncated = %q, want 'depth limit exceeded'", id.Truncated)
		}
		// StringType is not an ObjectType/DiscriminatedObjectType, so typeStr is used.
		if id.Type != "StringType" {
			t.Errorf("id.Type = %q, want StringType", id.Type)
		}
	})

	t.Run("DiscriminatedObjectType as nested property truncated with object type and name", func(t *testing.T) {
		strPtr := func(s string) *string { return &s }
		localTypes := TypesFile{
			// [0] StringType
			{Type: "StringType"},
			// [1] DiscriminatedObjectType "Policies"
			{
				Type:          "DiscriminatedObjectType",
				Name:          strPtr("Policies"),
				Discriminator: strPtr("kind"),
				BaseProperties: map[string]PropertyDef{
					"kind": {Type: Ref{Ref: "#/0"}, Flags: FlagRequired},
				},
				ElementMap: map[string]Ref{},
			},
			// [2] ObjectType "Root" with a property pointing to the DiscriminatedObjectType
			{
				Type: "ObjectType",
				Name: strPtr("Root"),
				Properties: map[string]PropertyDef{
					"policy": {Type: Ref{Ref: "#/1"}, Flags: 0},
				},
			},
			// [3] ResourceType body -> #/2
			{Type: "ResourceType", Body: &Ref{Ref: "#/2"}},
		}
		r := NewResolver(localTypes, 0)
		resolved, err := r.ResolveResourceType(3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		policy := resolved.Properties["policy"]
		if policy == nil {
			t.Fatal("policy property missing")
		}
		// The DiscriminatedObjectType at depth 1 exceeds maxDepth=0.
		if policy.Truncated != "depth limit exceeded" {
			t.Errorf("policy.Truncated = %q, want 'depth limit exceeded'", policy.Truncated)
		}
		// Should have Type="object" and Name="Policies", not Type="DiscriminatedObjectType".
		if policy.Type != "object" {
			t.Errorf("policy.Type = %q, want object", policy.Type)
		}
		if policy.Name != "Policies" {
			t.Errorf("policy.Name = %q, want Policies", policy.Name)
		}
	})

	t.Run("DiscriminatedObjectType oneOf is in stable alphabetical order", func(t *testing.T) {
		r := NewResolver(types, 5)
		resolved, err := r.ResolveResourceType(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resolved.OneOf) != 2 {
			t.Fatalf("OneOf length = %d, want 2", len(resolved.OneOf))
		}
		// Keys are "ftp" and "scm" — alphabetical order means ftp comes first.
		if resolved.OneOf[0].Name != "FtpPolicy" {
			t.Errorf("OneOf[0].Name = %q, want FtpPolicy", resolved.OneOf[0].Name)
		}
		if resolved.OneOf[1].Name != "ScmPolicy" {
			t.Errorf("OneOf[1].Name = %q, want ScmPolicy", resolved.OneOf[1].Name)
		}
	})
}

func TestDiscriminatedObjectType(t *testing.T) {
	types := buildTestTypes()
	r := NewResolver(types, 5)

	t.Run("resolve includes baseProperties", func(t *testing.T) {
		resolved, err := r.ResolveResourceType(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resolved.Type != "object" {
			t.Errorf("Type = %q, want object", resolved.Type)
		}
		if resolved.Name != "PublishingPolicies" {
			t.Errorf("Name = %q, want PublishingPolicies", resolved.Name)
		}
		id, ok := resolved.Properties["id"]
		if !ok {
			t.Fatal("expected 'id' in baseProperties")
		}
		if id.Type != "string" {
			t.Errorf("id.Type = %q, want string", id.Type)
		}
		if id.ReadOnly == nil || !*id.ReadOnly {
			t.Error("id should be readOnly")
		}
	})

	t.Run("resolve includes discriminated variants as oneOf", func(t *testing.T) {
		resolved, err := r.ResolveResourceType(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resolved.OneOf) != 2 {
			t.Fatalf("OneOf length = %d, want 2", len(resolved.OneOf))
		}
		// Both variants should be object types (FtpPolicy and ScmPolicy).
		for _, variant := range resolved.OneOf {
			if variant.Type != "object" {
				t.Errorf("variant.Type = %q, want object", variant.Type)
			}
		}
	})

	t.Run("TypeString returns name", func(t *testing.T) {
		got := r.TypeString(&types[19])
		if got != "PublishingPolicies" {
			t.Errorf("TypeString = %q, want PublishingPolicies", got)
		}
	})

	t.Run("TypeString without name returns object", func(t *testing.T) {
		unnamed := TypeEntry{Type: "DiscriminatedObjectType"}
		got := r.TypeString(&unnamed)
		if got != "object" {
			t.Errorf("TypeString = %q, want object", got)
		}
	})

	t.Run("unmarshal DiscriminatedObjectType from types file", func(t *testing.T) {
		// Mirrors the real bicep-types-az format that triggered the original bug.
		data := []byte(`[
			{"$type":"StringType"},
			{"$type":"StringLiteralType","value":"ftp"},
			{"$type":"StringLiteralType","value":"scm"},
			{
				"$type":"DiscriminatedObjectType",
				"name":"microsoft.web/sites/basicpublishingcredentialspolicies",
				"discriminator":"name",
				"baseProperties":{},
				"elements":{
					"ftp":{"$ref":"#/1"},
					"scm":{"$ref":"#/2"}
				}
			},
			{"$type":"ResourceType","name":"Microsoft.Web/sites/basicPublishingCredentialsPolicies@2019-08-01","body":{"$ref":"#/3"}}
		]`)
		tf, err := ParseTypesFile(data)
		if err != nil {
			t.Fatalf("ParseTypesFile: %v", err)
		}
		if len(tf) != 5 {
			t.Fatalf("len = %d, want 5", len(tf))
		}
		dot := tf[3]
		if dot.Type != "DiscriminatedObjectType" {
			t.Errorf("Type = %q, want DiscriminatedObjectType", dot.Type)
		}
		if len(dot.Elements) != 0 {
			t.Errorf("Elements should be empty, got %d", len(dot.Elements))
		}
		if len(dot.ElementMap) != 2 {
			t.Fatalf("ElementMap length = %d, want 2", len(dot.ElementMap))
		}
		if dot.ElementMap["ftp"].Ref != "#/1" {
			t.Errorf("ElementMap[ftp] = %q, want #/1", dot.ElementMap["ftp"].Ref)
		}
		if dot.ElementMap["scm"].Ref != "#/2" {
			t.Errorf("ElementMap[scm] = %q, want #/2", dot.ElementMap["scm"].Ref)
		}
	})
}
