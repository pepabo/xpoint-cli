package xpoint

import (
	"strings"
	"testing"
)

func TestSchemaAliases_Sorted(t *testing.T) {
	got := SchemaAliases()
	want := []string{"approval.list", "document.search", "form.list"}
	if len(got) != len(want) {
		t.Fatalf("aliases = %v", got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("aliases[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestLookupOperation_Unknown(t *testing.T) {
	_, err := LookupOperation("nope.missing", false)
	if err == nil || !strings.Contains(err.Error(), "unknown schema alias") {
		t.Errorf("err = %v", err)
	}
}

func TestLookupOperation_FormList_NoResolve(t *testing.T) {
	op, err := LookupOperation("form.list", false)
	if err != nil {
		t.Fatalf("LookupOperation: %v", err)
	}
	if op["_method"] != "GET" {
		t.Errorf("_method = %v", op["_method"])
	}
	if op["_path"] != "/api/v1/forms" {
		t.Errorf("_path = %v", op["_path"])
	}
	if op["operationId"] != "GetAvailableFormList" {
		t.Errorf("operationId = %v", op["operationId"])
	}
	params, ok := op["parameters"].([]any)
	if !ok || len(params) == 0 {
		t.Fatalf("parameters = %v", op["parameters"])
	}
	// Without --resolve-refs, the first parameter is still a $ref.
	first, _ := params[0].(map[string]any)
	if _, hasRef := first["$ref"]; !hasRef {
		t.Errorf("expected $ref in parameter, got %v", first)
	}
}

func TestLookupOperation_FormList_ResolveRefs(t *testing.T) {
	op, err := LookupOperation("form.list", true)
	if err != nil {
		t.Fatalf("LookupOperation: %v", err)
	}
	params, _ := op["parameters"].([]any)
	if len(params) == 0 {
		t.Fatalf("no parameters")
	}
	first, _ := params[0].(map[string]any)
	if _, hasRef := first["$ref"]; hasRef {
		t.Errorf("expected $ref to be inlined, got %v", first)
	}
	if first["name"] != "fgid" {
		t.Errorf("first param name = %v", first["name"])
	}
}

func TestLookupOperation_DocumentSearch(t *testing.T) {
	op, err := LookupOperation("document.search", false)
	if err != nil {
		t.Fatalf("LookupOperation: %v", err)
	}
	if op["_method"] != "POST" {
		t.Errorf("_method = %v", op["_method"])
	}
	if op["_path"] != "/api/v1/search/documents" {
		t.Errorf("_path = %v", op["_path"])
	}
}

func TestLookupRef_MissingSegment(t *testing.T) {
	root := map[string]any{"a": map[string]any{"b": 1}}
	if _, err := lookupRef(root, "#/a/c"); err == nil {
		t.Error("expected error for missing segment")
	}
}
