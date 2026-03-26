package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestFixture creates a local directory structure that mimics
// the bicep-types-az generated directory for integration testing.
func buildTestFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create index.json
	indexJSON := `{
		"resources": {
			"Microsoft.Test/resources@2023-01-01": {"$ref": "test/microsoft.test/2023-01-01/types.json#/8"},
			"Microsoft.Test/resources@2024-01-01": {"$ref": "test/microsoft.test/2024-01-01/types.json#/8"},
			"Microsoft.Test/resources/children@2023-01-01": {"$ref": "test/microsoft.test/2023-01-01/types.json#/8"},
			"Microsoft.Other/things@2023-01-01": {"$ref": "other/microsoft.other/2023-01-01/types.json#/5"}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(indexJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create types.json for Microsoft.Test
	typesDir := filepath.Join(dir, "test", "microsoft.test", "2023-01-01")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	typesJSON := `[
		{"$type":"StringType"},
		{"$type":"StringLiteralType","value":"Active"},
		{"$type":"StringLiteralType","value":"Inactive"},
		{"$type":"UnionType","elements":[{"$ref":"#/1"},{"$ref":"#/2"},{"$ref":"#/0"}]},
		{"$type":"StringLiteralType","value":"Microsoft.Test/resources"},
		{"$type":"StringLiteralType","value":"2023-01-01"},
		{"$type":"ObjectType","name":"TestProperties","properties":{"status":{"type":{"$ref":"#/3"},"flags":0,"description":"The provisioning status"}}},
		{"$type":"ObjectType","name":"Microsoft.Test/resources","properties":{"id":{"type":{"$ref":"#/0"},"flags":10,"description":"The resource id"},"name":{"type":{"$ref":"#/0"},"flags":9,"description":"The resource name"},"type":{"type":{"$ref":"#/4"},"flags":10,"description":"The resource type"},"apiVersion":{"type":{"$ref":"#/5"},"flags":10,"description":"The resource api version"},"properties":{"type":{"$ref":"#/6"},"flags":0,"description":"Resource properties"}}},
		{"$type":"ResourceType","name":"Microsoft.Test/resources@2023-01-01","body":{"$ref":"#/7"}}
	]`
	if err := os.WriteFile(filepath.Join(typesDir, "types.json"), []byte(typesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create types.json for 2024 version (same content, different dir)
	typesDir2 := filepath.Join(dir, "test", "microsoft.test", "2024-01-01")
	if err := os.MkdirAll(typesDir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir2, "types.json"), []byte(typesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create types.json for Microsoft.Other
	otherDir := filepath.Join(dir, "other", "microsoft.other", "2023-01-01")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherJSON := `[
		{"$type":"StringType"},
		{"$type":"BooleanType"},
		{"$type":"StringLiteralType","value":"Microsoft.Other/things"},
		{"$type":"StringLiteralType","value":"2023-01-01"},
		{"$type":"ObjectType","name":"Microsoft.Other/things","properties":{"id":{"type":{"$ref":"#/0"},"flags":10,"description":"The resource id"},"name":{"type":{"$ref":"#/0"},"flags":9,"description":"The resource name"},"enabled":{"type":{"$ref":"#/1"},"flags":1,"description":"Whether enabled"}}},
		{"$type":"ResourceType","name":"Microsoft.Other/things@2023-01-01","body":{"$ref":"#/4"}}
	]`
	if err := os.WriteFile(filepath.Join(otherDir, "types.json"), []byte(otherJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// runCLI runs the CLI app with given args and a --types-dir pointing to the fixture.
// Returns stdout output and error.
func runCLI(t *testing.T, fixtureDir string, args ...string) (string, error) {
	t.Helper()

	// Build args: azureschema --types-dir <fixture> <args...>
	fullArgs := []string{"azureschema", "--types-dir", fixtureDir}
	fullArgs = append(fullArgs, args...)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture stderr
	oldStderr := os.Stderr
	_, wErr, _ := os.Pipe()
	os.Stderr = wErr

	app := buildApp()

	err := app.Run(t.Context(), fullArgs)

	w.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	return string(buf[:n]), err
}

// --- Integration tests ---

func TestCLIGetSummary(t *testing.T) {
	fixture := buildTestFixture(t)
	output, err := runCLI(t, fixture, "get", "Microsoft.Test/resources", "2023-01-01")
	if err != nil {
		t.Fatalf("CLI error: %v", err)
	}

	if !strings.Contains(output, "Microsoft.Test/resources @ 2023-01-01") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "PROPERTIES:") {
		t.Error("missing PROPERTIES section")
	}
	if !strings.Contains(output, "id: string") {
		t.Error("missing id property")
	}
	if !strings.Contains(output, "[READ-ONLY]") {
		t.Error("missing READ-ONLY flag")
	}
	if !strings.Contains(output, "[REQUIRED]") {
		t.Error("missing REQUIRED flag")
	}
	if !strings.Contains(output, "Required: name") {
		t.Error("missing Required line")
	}
}

func TestCLIGetJSON(t *testing.T) {
	fixture := buildTestFixture(t)
	output, err := runCLI(t, fixture, "get", "Microsoft.Test/resources", "2023-01-01", "--json")
	if err != nil {
		t.Fatalf("CLI error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if parsed["type"] != "object" {
		t.Errorf("type = %v, want object", parsed["type"])
	}
}

func TestCLIGetDepth(t *testing.T) {
	fixture := buildTestFixture(t)
	output, err := runCLI(t, fixture, "get", "Microsoft.Test/resources", "2023-01-01", "--depth", "0")
	if err != nil {
		t.Fatalf("CLI error: %v", err)
	}

	// At depth 0, nested properties should show truncation
	if !strings.Contains(output, "...depth limit exceeded") {
		t.Error("expected depth limit exceeded message")
	}
}

func TestCLIGetCaseInsensitive(t *testing.T) {
	fixture := buildTestFixture(t)
	output, err := runCLI(t, fixture, "get", "microsoft.test/resources", "2023-01-01")
	if err != nil {
		t.Fatalf("CLI error: %v", err)
	}

	if !strings.Contains(output, "PROPERTIES:") {
		t.Error("case-insensitive lookup failed")
	}
}

func TestCLIGetNotFound(t *testing.T) {
	fixture := buildTestFixture(t)
	_, err := runCLI(t, fixture, "get", "Microsoft.Fake/nonexistent", "2023-01-01")
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, should contain 'not found'", err)
	}
}

func TestCLIGetMissingArgs(t *testing.T) {
	fixture := buildTestFixture(t)
	_, err := runCLI(t, fixture, "get")
	if err == nil {
		t.Fatal("expected error for missing arguments")
	}
}

func TestCLIVersions(t *testing.T) {
	fixture := buildTestFixture(t)
	output, err := runCLI(t, fixture, "versions", "Microsoft.Test")
	if err != nil {
		t.Fatalf("CLI error: %v", err)
	}

	if !strings.Contains(output, "Microsoft.Test/resources") {
		t.Error("missing resource type in versions output")
	}
	if !strings.Contains(output, "2023-01-01") {
		t.Error("missing 2023-01-01 version")
	}
	if !strings.Contains(output, "2024-01-01") {
		t.Error("missing 2024-01-01 version")
	}
}

func TestCLIVersionsNoMatch(t *testing.T) {
	fixture := buildTestFixture(t)
	_, err := runCLI(t, fixture, "versions", "Microsoft.Nonexistent")
	if err == nil {
		t.Fatal("expected error for no matching provider")
	}
}

func TestCLIVersionsMissingArgs(t *testing.T) {
	fixture := buildTestFixture(t)
	_, err := runCLI(t, fixture, "versions")
	if err == nil {
		t.Fatal("expected error for missing arguments")
	}
}
