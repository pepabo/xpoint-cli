package xpoint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

//go:embed schema.json
var schemaJSON []byte

var (
	schemaOnce    sync.Once
	schemaOps     map[string]any
	schemaOpsKeys []string
	schemaErr     error
)

func loadSchema() (map[string]any, []string, error) {
	schemaOnce.Do(func() {
		var root map[string]any
		if err := json.Unmarshal(schemaJSON, &root); err != nil {
			schemaErr = fmt.Errorf("parse embedded schema.json: %w", err)
			return
		}
		ops, ok := root["operations"].(map[string]any)
		if !ok {
			schemaErr = fmt.Errorf("schema.json is missing `operations` mapping")
			return
		}
		keys := make([]string, 0, len(ops))
		for k := range ops {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		schemaOps = ops
		schemaOpsKeys = keys
	})
	return schemaOps, schemaOpsKeys, schemaErr
}

// SchemaAliases returns the sorted list of supported dotted aliases.
func SchemaAliases() []string {
	_, keys, err := loadSchema()
	if err != nil {
		return nil
	}
	out := make([]string, len(keys))
	copy(out, keys)
	return out
}

// LookupOperation returns the schema object for the given alias.
func LookupOperation(alias string) (map[string]any, error) {
	ops, _, err := loadSchema()
	if err != nil {
		return nil, err
	}
	op, ok := ops[alias].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unknown schema alias %q (run `xp schema` to list supported aliases)", alias)
	}
	return op, nil
}
