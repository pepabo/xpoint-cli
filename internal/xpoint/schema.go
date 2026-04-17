package xpoint

import (
	_ "embed"
	"fmt"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed schema.yaml
var schemaYAML []byte

var (
	schemaOnce    sync.Once
	schemaOps     map[string]any
	schemaOpsKeys []string
	schemaErr     error
)

func loadSchema() (map[string]any, []string, error) {
	schemaOnce.Do(func() {
		var raw any
		if err := yaml.Unmarshal(schemaYAML, &raw); err != nil {
			schemaErr = fmt.Errorf("parse embedded schema.yaml: %w", err)
			return
		}
		root, ok := toStringKeyed(raw).(map[string]any)
		if !ok {
			schemaErr = fmt.Errorf("schema.yaml root is not a mapping")
			return
		}
		ops, ok := root["operations"].(map[string]any)
		if !ok {
			schemaErr = fmt.Errorf("schema.yaml is missing `operations` mapping")
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

// toStringKeyed recursively converts yaml.v3's map[any]any / map[string]any
// trees into purely JSON-compatible forms with string keys.
func toStringKeyed(v any) any {
	switch m := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprint(k)] = toStringKeyed(val)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = toStringKeyed(val)
		}
		return out
	case []any:
		out := make([]any, len(m))
		for i, val := range m {
			out[i] = toStringKeyed(val)
		}
		return out
	default:
		return v
	}
}
