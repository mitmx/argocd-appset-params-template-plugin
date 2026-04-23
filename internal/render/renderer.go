package render

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/maps"
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

func (r Renderer) Resolve(result map[string]any, contextBuilder func(current map[string]any) map[string]any) (map[string]any, error) {
	current := maps.CloneMap(result)
	for pass := 1; pass <= r.MaxPasses; pass++ {
		nextAny, changed, err := renderAny(current, contextBuilder(current), "result")
		if err != nil {
			return nil, err
		}
		next, ok := nextAny.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("internal error: rendered root is %T, want map[string]any", nextAny)
		}

		stable := !changed || reflect.DeepEqual(current, next)
		if stable {
			if path, ok := findUnresolvedTemplatePath(next, "result"); ok {
				return nil, fmt.Errorf("unresolved template remains at %s", path)
			}
			return next, nil
		}
		current = next
	}
	if path, ok := findUnresolvedTemplatePath(current, "result"); ok {
		return nil, fmt.Errorf("templates did not stabilize after %d passes; last unresolved path: %s", r.MaxPasses, path)
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
		expr, isPureAction := pureActionExpression(t)
		if isPureAction {
			rendered, err := renderTypedExpression(expr, ctx)
			if err != nil {
				return nil, false, fmt.Errorf("render %s: %w", path, err)
			}
			return rendered, !reflect.DeepEqual(rendered, t), nil
		}
		if !looksLikeTemplate(t) {
			return t, false, nil
		}
		rendered, err := renderString(t, ctx)
		if err != nil {
			return nil, false, fmt.Errorf("render %s: %w", path, err)
		}
		return rendered, rendered != t, nil
	default:
		return value, false, nil
	}
}

func renderTypedExpression(expr string, ctx map[string]any) (any, error) {
	wrapped := "{{ " + expr + " | __typed__ }}"
	rendered, err := renderString(wrapped, ctx)
	if err != nil {
		return nil, err
	}
	return decodeJSONValue(rendered)
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

func pureActionExpression(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "{{") || !strings.HasSuffix(trimmed, "}}") {
		return "", false
	}
	inner := strings.TrimSpace(trimmed[2 : len(trimmed)-2])
	if strings.HasPrefix(inner, "-") {
		inner = strings.TrimSpace(inner[1:])
	}
	if strings.HasSuffix(inner, "-") {
		inner = strings.TrimSpace(inner[:len(inner)-1])
	}
	if inner == "" {
		return "", false
	}
	if strings.Contains(inner, "{{") || strings.Contains(inner, "}}") {
		return "", false
	}
	return inner, true
}

func looksLikeTemplate(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

func findUnresolvedTemplatePath(value any, path string) (string, bool) {
	switch t := value.(type) {
	case map[string]any:
		for _, key := range maps.SortedKeys(t) {
			if found, ok := findUnresolvedTemplatePath(t[key], joinPath(path, key)); ok {
				return found, true
			}
		}
		return "", false
	case []any:
		for i := range t {
			if found, ok := findUnresolvedTemplatePath(t[i], fmt.Sprintf("%s[%d]", path, i)); ok {
				return found, true
			}
		}
		return "", false
	case string:
		if looksLikeTemplate(t) {
			return path, true
		}
		return "", false
	default:
		return "", false
	}
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
