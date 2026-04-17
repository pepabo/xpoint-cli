package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchemaCmd_ListsAliases(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runSchema(schemaCmd, nil)
	})
	if err != nil {
		t.Fatalf("runSchema: %v", err)
	}
	for _, want := range []string{"form.list", "approval.list", "document.search"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestSchemaCmd_EmitsJSON(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runSchema(schemaCmd, []string{"form.list"})
	})
	if err != nil {
		t.Fatalf("runSchema: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output not JSON: %v (%s)", err, out)
	}
	if decoded["method"] != "GET" || decoded["path"] != "/api/v1/forms" {
		t.Errorf("decoded = %v", decoded)
	}
}

func TestSchemaCmd_UnknownAlias(t *testing.T) {
	err := runSchema(schemaCmd, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown schema alias") {
		t.Errorf("err = %v", err)
	}
}
