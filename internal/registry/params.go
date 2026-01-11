package registry

import (
	"fmt"
	"regexp"
	"strings"
)

var paramRe = regexp.MustCompile(`{{\s*([a-zA-Z0-9_.-]+)\s*}}`)

// FindParams returns a unique list of parameter names referenced in s in order of appearance.
func FindParams(s string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, m := range paramRe.FindAllStringSubmatch(s, -1) {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// ApplyParams replaces parameter placeholders in s using values from params.
// If a parameter is missing, an error is returned listing missing keys.
func ApplyParams(s string, params map[string]string) (string, error) {
	missing := []string{}
	result := paramRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := paramRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		name := sub[1]
		if v, ok := params[name]; ok {
			return v
		}
		missing = append(missing, name)
		return match
	})
	if len(missing) > 0 {
		// dedupe
		uniq := map[string]bool{}
		for _, m := range missing {
			uniq[m] = true
		}
		keys := []string{}
		for k := range uniq {
			keys = append(keys, k)
		}
		return result, fmt.Errorf("missing parameters: %s", strings.Join(keys, ", "))
	}
	return result, nil
}