// Package schema serves xpoint-cli's curated operation schemas.
// Each supported operation is stored as its own JSON file in this
// directory (e.g. form.list.json) and embedded into the binary.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

//go:embed *.json
var files embed.FS

var (
	loadOnce sync.Once
	ops      map[string]map[string]any
	aliases  []string
	loadErr  error
)

func load() (map[string]map[string]any, []string, error) {
	loadOnce.Do(func() {
		entries, err := fs.ReadDir(files, ".")
		if err != nil {
			loadErr = fmt.Errorf("read embedded schema dir: %w", err)
			return
		}
		ops = make(map[string]map[string]any, len(entries))
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			alias := strings.TrimSuffix(e.Name(), ".json")
			data, err := fs.ReadFile(files, e.Name())
			if err != nil {
				loadErr = fmt.Errorf("read %s: %w", e.Name(), err)
				return
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				loadErr = fmt.Errorf("parse %s: %w", e.Name(), err)
				return
			}
			ops[alias] = m
		}
		aliases = make([]string, 0, len(ops))
		for k := range ops {
			aliases = append(aliases, k)
		}
		sort.Strings(aliases)
	})
	return ops, aliases, loadErr
}

// Aliases returns the sorted list of supported dotted aliases.
func Aliases() []string {
	_, a, err := load()
	if err != nil {
		return nil
	}
	out := make([]string, len(a))
	copy(out, a)
	return out
}

// Lookup returns the schema object for the given alias.
func Lookup(alias string) (map[string]any, error) {
	o, _, err := load()
	if err != nil {
		return nil, err
	}
	op, ok := o[alias]
	if !ok {
		return nil, fmt.Errorf("unknown schema alias %q (run `xp schema` to list supported aliases)", alias)
	}
	return op, nil
}
