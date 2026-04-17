package xpoint

import (
	_ "embed"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openapiYAML []byte

// Operation maps a dotted schema path (e.g. "form.list") to an OpenAPI
// operation in the embedded X-point spec.
type Operation struct {
	Path   string
	Method string
}

// operationAliases maps user-facing identifiers to OpenAPI path+method pairs.
// The set mirrors the commands exposed by the CLI.
var operationAliases = map[string]Operation{
	"form.list":       {Path: "/api/v1/forms", Method: "get"},
	"approval.list":   {Path: "/api/v1/approvals", Method: "get"},
	"document.search": {Path: "/api/v1/search/documents", Method: "post"},
}

var (
	specOnce sync.Once
	specRoot map[string]any
	specErr  error
)

func loadSpec() (map[string]any, error) {
	specOnce.Do(func() {
		var raw any
		if err := yaml.Unmarshal(openapiYAML, &raw); err != nil {
			specErr = fmt.Errorf("parse embedded openapi.yaml: %w", err)
			return
		}
		norm, ok := toStringKeyed(raw).(map[string]any)
		if !ok {
			specErr = fmt.Errorf("openapi root is not a mapping")
			return
		}
		specRoot = norm
	})
	return specRoot, specErr
}

// SchemaAliases returns the sorted list of supported dotted aliases.
func SchemaAliases() []string {
	out := make([]string, 0, len(operationAliases))
	for k := range operationAliases {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LookupOperation returns the OpenAPI operation object for the given alias.
// If resolveRefs is true, all $ref pointers within the operation are
// recursively inlined.
func LookupOperation(alias string, resolveRefs bool) (map[string]any, error) {
	op, ok := operationAliases[alias]
	if !ok {
		return nil, fmt.Errorf("unknown schema alias %q (run `xp schema` to list supported aliases)", alias)
	}
	root, err := loadSpec()
	if err != nil {
		return nil, err
	}
	paths, _ := root["paths"].(map[string]any)
	if paths == nil {
		return nil, fmt.Errorf("openapi spec is missing `paths`")
	}
	path, _ := paths[op.Path].(map[string]any)
	if path == nil {
		return nil, fmt.Errorf("openapi spec is missing path %q", op.Path)
	}
	opObj, _ := path[op.Method].(map[string]any)
	if opObj == nil {
		return nil, fmt.Errorf("openapi spec is missing %s %s", strings.ToUpper(op.Method), op.Path)
	}
	result := cloneMap(opObj)
	result["_path"] = op.Path
	result["_method"] = strings.ToUpper(op.Method)
	if resolveRefs {
		resolved, err := resolveRefsIn(result, root, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		m, _ := resolved.(map[string]any)
		return m, nil
	}
	return result, nil
}

func resolveRefsIn(node any, root map[string]any, seen map[string]bool) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok && len(v) == 1 {
			if seen[ref] {
				return map[string]any{"$ref": ref, "_circular": true}, nil
			}
			target, err := lookupRef(root, ref)
			if err != nil {
				return nil, err
			}
			next := make(map[string]bool, len(seen)+1)
			for k := range seen {
				next[k] = true
			}
			next[ref] = true
			return resolveRefsIn(target, root, next)
		}
		out := make(map[string]any, len(v))
		for k, child := range v {
			resolved, err := resolveRefsIn(child, root, seen)
			if err != nil {
				return nil, err
			}
			out[k] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			resolved, err := resolveRefsIn(child, root, seen)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return v, nil
	}
}

// lookupRef resolves a local JSON pointer like "#/components/schemas/Foo".
func lookupRef(root map[string]any, ref string) (any, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("external or non-fragment $ref not supported: %q", ref)
	}
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var cur any = root
	for _, raw := range parts {
		seg, err := url.PathUnescape(raw)
		if err != nil {
			seg = raw
		}
		seg = strings.ReplaceAll(strings.ReplaceAll(seg, "~1", "/"), "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot resolve %q: %q is not a mapping", ref, seg)
		}
		child, ok := m[seg]
		if !ok {
			return nil, fmt.Errorf("cannot resolve %q: segment %q not found", ref, seg)
		}
		cur = child
	}
	return cur, nil
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
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
