package render

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/mitmx/argocd-values-pipeline-plugin/internal/maps"
)

type Renderer struct {
	MaxPasses int
}

func New(maxPasses int) Renderer {
	if maxPasses <= 0 {
		maxPasses = 4
	}
	return Renderer{MaxPasses: maxPasses}
}

func (r Renderer) Resolve(values map[string]any, contextBuilder func(current map[string]any) map[string]any) (map[string]any, error) {
	current := maps.CloneMap(values)
	for pass := 1; pass <= r.MaxPasses; pass++ {
		nextAny, changed, err := renderAny(current, contextBuilder(current), "values")
		if err != nil {
			return nil, err
		}
		next, ok := nextAny.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("internal error: rendered root is %T, want map[string]any", nextAny)
		}
		if !changed || reflect.DeepEqual(current, next) {
			return next, nil
		}
		current = next
	}
	return nil, fmt.Errorf("templates did not stabilize after %d passes", r.MaxPasses)
}

func renderAny(value any, ctx map[string]any, path string) (any, bool, error) {
	switch t := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		changed := false
		for _, key := range maps.SortedKeys(t) {
			next, itemChanged, err := renderAny(t[key], ctx, joinPath(path, key))
			if err != nil {
				return nil, false, err
			}
			out[key] = next
			changed = changed || itemChanged
		}
		return out, changed, nil
	case []any:
		out := make([]any, len(t))
		changed := false
		for i := range t {
			next, itemChanged, err := renderAny(t[i], ctx, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, false, err
			}
			out[i] = next
			changed = changed || itemChanged
		}
		return out, changed, nil
	case string:
		if !looksLikeTemplate(t) {
			return t, false, nil
		}
		out, err := renderString(t, ctx)
		if err != nil {
			return nil, false, fmt.Errorf("render %s: %w", path, err)
		}
		return out, out != t, nil
	default:
		return value, false, nil
	}
}

func renderString(src string, ctx map[string]any) (string, error) {
	tpl, err := template.New("value").Option("missingkey=error").Funcs(FuncMap()).Parse(src)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, ctx); err != nil {
		return "", err
	}
	return b.String(), nil
}

func looksLikeTemplate(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
