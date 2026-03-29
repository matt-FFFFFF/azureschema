package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/matt-FFFFFF/azureschema/internal/bicep"
)

// buildTestTypes creates a small set of types for render testing.
// Layout:
//
//	[0] StringType
//	[1] StringLiteralType "2023-01-01"
//	[2] StringLiteralType "Microsoft.Test/resources"
//	[3] BooleanType
//	[4] IntegerType
//	[5] StringLiteralType "Active"
//	[6] StringLiteralType "Inactive"
//	[7] UnionType [#/5, #/6, #/0]
//	[8] ArrayType (itemType -> #/0)
//	[9] ObjectType "InnerProps" {status: #/7 (flags=0, desc="The status"), enabled: #/3 (flags=1, desc="Is enabled")}
//	[10] ObjectType "TestResource" {id: #/0 (flags=10), name: #/0 (flags=9), type: #/2 (flags=10), apiVersion: #/1 (flags=10), properties: #/9 (flags=0), tags: #/8 (flags=4)}
//	[11] ResourceType body -> #/10
func buildTestTypes() bicep.TypesFile {
	strPtr := func(s string) *string { return &s }
	return bicep.TypesFile{
		{Type: "StringType"}, // 0
		{Type: "StringLiteralType", Value: strPtr("2023-01-01")},               // 1
		{Type: "StringLiteralType", Value: strPtr("Microsoft.Test/resources")}, // 2
		{Type: "BooleanType"},                                  // 3
		{Type: "IntegerType"},                                  // 4
		{Type: "StringLiteralType", Value: strPtr("Active")},   // 5
		{Type: "StringLiteralType", Value: strPtr("Inactive")}, // 6
		{Type: "UnionType", Elements: []bicep.Ref{{Ref: "#/5"}, {Ref: "#/6"}, {Ref: "#/0"}}}, // 7
		{Type: "ArrayType", ItemType: &bicep.Ref{Ref: "#/0"}},                                // 8
		{
			Type: "ObjectType",
			Name: strPtr("InnerProps"),
			Properties: map[string]bicep.PropertyDef{
				"status":  {Type: bicep.Ref{Ref: "#/7"}, Flags: 0, Description: "The status"},
				"enabled": {Type: bicep.Ref{Ref: "#/3"}, Flags: bicep.FlagRequired, Description: "Is enabled"},
			},
		}, // 9
		{
			Type: "ObjectType",
			Name: strPtr("TestResource"),
			Properties: map[string]bicep.PropertyDef{
				"id":         {Type: bicep.Ref{Ref: "#/0"}, Flags: bicep.FlagReadOnly | bicep.FlagDeployTimeConst, Description: "The resource id"},
				"name":       {Type: bicep.Ref{Ref: "#/0"}, Flags: bicep.FlagRequired | bicep.FlagDeployTimeConst, Description: "The resource name"},
				"type":       {Type: bicep.Ref{Ref: "#/2"}, Flags: bicep.FlagReadOnly | bicep.FlagDeployTimeConst, Description: "The resource type"},
				"apiVersion": {Type: bicep.Ref{Ref: "#/1"}, Flags: bicep.FlagReadOnly | bicep.FlagDeployTimeConst, Description: "The resource api version"},
				"properties": {Type: bicep.Ref{Ref: "#/9"}, Flags: 0, Description: "Resource properties"},
				"tags":       {Type: bicep.Ref{Ref: "#/8"}, Flags: bicep.FlagWriteOnly, Description: "Resource tags"},
			},
		}, // 10
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Test/resources@2023-01-01"),
			Body: &bicep.Ref{Ref: "#/10"},
		}, // 11
	}
}

// --- JSON renderer tests ---

func TestJSON(t *testing.T) {
	types := buildTestTypes()
	resolver := bicep.NewResolver(types, 5)

	t.Run("produces valid JSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := JSON(&buf, resolver, 11)
		if err != nil {
			t.Fatalf("JSON: %v", err)
		}

		// Must be valid JSON.
		var parsed map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}
	})

	t.Run("root type is object", func(t *testing.T) {
		var buf bytes.Buffer
		if err := JSON(&buf, resolver, 11); err != nil {
			t.Fatal(err)
		}

		var parsed map[string]interface{}
		json.Unmarshal(buf.Bytes(), &parsed)

		if parsed["type"] != "object" {
			t.Errorf("root type = %v, want object", parsed["type"])
		}
		if parsed["name"] != "TestResource" {
			t.Errorf("root name = %v, want TestResource", parsed["name"])
		}
	})

	t.Run("properties present", func(t *testing.T) {
		var buf bytes.Buffer
		JSON(&buf, resolver, 11)

		var parsed map[string]interface{}
		json.Unmarshal(buf.Bytes(), &parsed)

		props, ok := parsed["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("properties not found or not an object")
		}

		// Check id is read-only
		id, ok := props["id"].(map[string]interface{})
		if !ok {
			t.Fatal("id not found")
		}
		if id["type"] != "string" {
			t.Errorf("id.type = %v, want string", id["type"])
		}
		if id["readOnly"] != true {
			t.Errorf("id.readOnly = %v, want true", id["readOnly"])
		}

		// Check name is required
		name, ok := props["name"].(map[string]interface{})
		if !ok {
			t.Fatal("name not found")
		}
		if name["required"] != true {
			t.Errorf("name.required = %v, want true", name["required"])
		}

		// Check tags is write-only
		tags, ok := props["tags"].(map[string]interface{})
		if !ok {
			t.Fatal("tags not found")
		}
		if tags["writeOnly"] != true {
			t.Errorf("tags.writeOnly = %v, want true", tags["writeOnly"])
		}
	})

	t.Run("nested object resolved", func(t *testing.T) {
		var buf bytes.Buffer
		JSON(&buf, resolver, 11)

		var parsed map[string]interface{}
		json.Unmarshal(buf.Bytes(), &parsed)

		props := parsed["properties"].(map[string]interface{})
		inner := props["properties"].(map[string]interface{})

		if inner["type"] != "object" {
			t.Errorf("properties.type = %v, want object", inner["type"])
		}
		if inner["name"] != "InnerProps" {
			t.Errorf("properties.name = %v, want InnerProps", inner["name"])
		}

		innerProps := inner["properties"].(map[string]interface{})
		enabled := innerProps["enabled"].(map[string]interface{})
		if enabled["type"] != "boolean" {
			t.Errorf("enabled.type = %v, want boolean", enabled["type"])
		}
		if enabled["required"] != true {
			t.Errorf("enabled.required = %v, want true", enabled["required"])
		}
	})

	t.Run("error for invalid index", func(t *testing.T) {
		var buf bytes.Buffer
		err := JSON(&buf, resolver, 999)
		if err == nil {
			t.Fatal("expected error for out-of-range index")
		}
	})
}

// --- Summary renderer tests ---

func TestSummary(t *testing.T) {
	types := buildTestTypes()
	resolver := bicep.NewResolver(types, 5)

	t.Run("produces correct header", func(t *testing.T) {
		var buf bytes.Buffer
		err := Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		if err != nil {
			t.Fatalf("Summary: %v", err)
		}

		output := buf.String()

		// Check header
		if !strings.Contains(output, "Microsoft.Test/resources @ 2023-01-01") {
			t.Error("missing resource type header")
		}
		if !strings.Contains(output, "━") {
			t.Error("missing header line")
		}
	})

	t.Run("contains PROPERTIES section", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		output := buf.String()

		if !strings.Contains(output, "PROPERTIES:") {
			t.Error("missing PROPERTIES section")
		}
		if !strings.Contains(output, "───") {
			t.Error("missing separator line")
		}
	})

	t.Run("shows property names and types", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		output := buf.String()

		// id: string [READ-ONLY]
		if !strings.Contains(output, "id: string") {
			t.Error("missing id property")
		}
		if !strings.Contains(output, "[READ-ONLY]") {
			t.Error("missing READ-ONLY flag")
		}

		// name: string [REQUIRED]
		if !strings.Contains(output, "name: string") {
			t.Error("missing name property")
		}
		if !strings.Contains(output, "[REQUIRED]") {
			t.Error("missing REQUIRED flag")
		}

		// tags: array<string> [WRITE-ONLY]
		if !strings.Contains(output, "tags: array<string>") {
			t.Error("missing tags property")
		}
		if !strings.Contains(output, "[WRITE-ONLY]") {
			t.Error("missing WRITE-ONLY flag")
		}
	})

	t.Run("shows descriptions", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		output := buf.String()

		if !strings.Contains(output, "The resource id") {
			t.Error("missing description for id")
		}
		if !strings.Contains(output, "The resource name") {
			t.Error("missing description for name")
		}
	})

	t.Run("shows nested properties", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		output := buf.String()

		// Inner properties should be indented
		if !strings.Contains(output, "enabled: boolean") {
			t.Error("missing nested enabled property")
		}
		if !strings.Contains(output, "status:") {
			t.Error("missing nested status property")
		}
	})

	t.Run("shows required summary", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 11, "Microsoft.Test/resources", "2023-01-01")
		output := buf.String()

		if !strings.Contains(output, "Required: name") {
			t.Error("missing Required summary")
		}
	})

	t.Run("error for invalid index", func(t *testing.T) {
		var buf bytes.Buffer
		err := Summary(&buf, resolver, 999, "Fake", "2023-01-01")
		if err == nil {
			t.Fatal("expected error for out-of-range index")
		}
	})
}

func TestSummaryDepthLimit(t *testing.T) {
	// Create types with 3 levels of nesting:
	// TopLevel -> MidLevel -> DeepLevel {value: string}
	strPtr := func(s string) *string { return &s }
	types := bicep.TypesFile{
		{Type: "StringType"}, // 0
		{
			Type: "ObjectType",
			Name: strPtr("DeepLevel"),
			Properties: map[string]bicep.PropertyDef{
				"value": {Type: bicep.Ref{Ref: "#/0"}, Flags: 0, Description: "A value"},
			},
		}, // 1
		{
			Type: "ObjectType",
			Name: strPtr("MidLevel"),
			Properties: map[string]bicep.PropertyDef{
				"deep": {Type: bicep.Ref{Ref: "#/1"}, Flags: 0, Description: "Deep nested"},
			},
		}, // 2
		{
			Type: "ObjectType",
			Name: strPtr("TopLevel"),
			Properties: map[string]bicep.PropertyDef{
				"mid": {Type: bicep.Ref{Ref: "#/2"}, Flags: 0, Description: "Mid level"},
			},
		}, // 3
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Test/deep@2023-01-01"),
			Body: &bicep.Ref{Ref: "#/3"},
		}, // 4
	}

	t.Run("depth 1 truncates deep nesting", func(t *testing.T) {
		resolver := bicep.NewResolver(types, 1)
		var buf bytes.Buffer
		err := Summary(&buf, resolver, 4, "Microsoft.Test/deep", "2023-01-01")
		if err != nil {
			t.Fatalf("Summary: %v", err)
		}
		output := buf.String()

		// mid property should be shown
		if !strings.Contains(output, "mid: MidLevel") {
			t.Error("mid property should appear")
		}
		// deep property should be shown (depth=1 allows printing mid's children)
		if !strings.Contains(output, "deep: DeepLevel") {
			t.Error("deep property should appear at depth 1")
		}
		// But deep's children should be truncated
		if !strings.Contains(output, "...depth limit exceeded") {
			t.Error("should show depth limit exceeded for deep's children")
		}
	})

	t.Run("depth 5 shows everything", func(t *testing.T) {
		resolver := bicep.NewResolver(types, 5)
		var buf bytes.Buffer
		Summary(&buf, resolver, 4, "Microsoft.Test/deep", "2023-01-01")
		output := buf.String()

		if strings.Contains(output, "depth limit exceeded") {
			t.Error("should not show depth limit at depth 5")
		}
		if !strings.Contains(output, "value: string") {
			t.Error("deepest value property should appear")
		}
	})
}

func TestSummaryLongDescription(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	longDesc := strings.Repeat("x", 150)

	types := bicep.TypesFile{
		{Type: "StringType"}, // 0
		{
			Type: "ObjectType",
			Name: strPtr("Obj"),
			Properties: map[string]bicep.PropertyDef{
				"field": {Type: bicep.Ref{Ref: "#/0"}, Flags: 0, Description: longDesc},
			},
		}, // 1
		{
			Type: "ResourceType",
			Body: &bicep.Ref{Ref: "#/1"},
		}, // 2
	}

	resolver := bicep.NewResolver(types, 5)
	var buf bytes.Buffer
	Summary(&buf, resolver, 2, "Microsoft.Test/long", "2023-01-01")
	output := buf.String()

	// Description should be truncated to 120 chars + "..."
	if !strings.Contains(output, "...") {
		t.Error("long description should be truncated with ...")
	}
	// Full description should NOT appear
	if strings.Contains(output, longDesc) {
		t.Error("full long description should not appear")
	}
}

func TestSummaryDiscriminatedObjectType(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	types := bicep.TypesFile{
		// [0] StringType
		{Type: "StringType"},
		// [1] StringLiteralType "ftp"
		{Type: "StringLiteralType", Value: strPtr("ftp")},
		// [2] ObjectType "FtpPolicy"
		{
			Type: "ObjectType",
			Name: strPtr("FtpPolicy"),
			Properties: map[string]bicep.PropertyDef{
				"name": {Type: bicep.Ref{Ref: "#/1"}, Flags: bicep.FlagRequired, Description: "Policy name"},
			},
		},
		// [3] DiscriminatedObjectType "PublishingPolicies" with baseProperties
		{
			Type:          "DiscriminatedObjectType",
			Name:          strPtr("PublishingPolicies"),
			Discriminator: strPtr("name"),
			BaseProperties: map[string]bicep.PropertyDef{
				"id": {Type: bicep.Ref{Ref: "#/0"}, Flags: bicep.FlagReadOnly, Description: "The resource id"},
			},
			ElementMap: map[string]bicep.Ref{
				"ftp": {Ref: "#/2"},
			},
		},
		// [4] ResourceType body -> #/3
		{
			Type: "ResourceType",
			Name: strPtr("Microsoft.Web/sites/policies@2023-01-01"),
			Body: &bicep.Ref{Ref: "#/3"},
		},
	}
	resolver := bicep.NewResolver(types, 5)

	t.Run("shows baseProperties for DiscriminatedObjectType body", func(t *testing.T) {
		var buf bytes.Buffer
		err := Summary(&buf, resolver, 4, "Microsoft.Web/sites/policies", "2023-01-01")
		if err != nil {
			t.Fatalf("Summary: %v", err)
		}
		output := buf.String()

		// baseProperties should appear
		if !strings.Contains(output, "id: string") {
			t.Error("missing id property from baseProperties")
		}
		if !strings.Contains(output, "[READ-ONLY]") {
			t.Error("missing READ-ONLY flag for id")
		}
		if !strings.Contains(output, "The resource id") {
			t.Error("missing description for id")
		}
	})

	t.Run("Required summary excludes readOnly baseProperties", func(t *testing.T) {
		var buf bytes.Buffer
		Summary(&buf, resolver, 4, "Microsoft.Web/sites/policies", "2023-01-01")
		output := buf.String()

		// id is read-only, not required
		if strings.Contains(output, "Required: id") {
			t.Error("id should not appear in Required list")
		}
		// Should show empty required (no required props)
		if !strings.Contains(output, "Required:") {
			t.Error("missing Required section")
		}
	})
}
