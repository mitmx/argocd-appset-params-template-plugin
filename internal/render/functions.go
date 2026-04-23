package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/maps"
	"github.com/mitmx/argocd-appset-params-template-plugin/internal/serialize"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"default": func(fallback, value any) any {
			if maps.IsZero(value) {
				return fallback
			}
			return value
		},
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"trim":       strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"replace": func(old, new, src string) string {
			return strings.ReplaceAll(src, old, new)
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"quote": func(v any) (RawJSON, error) {
			b, err := json.Marshal(fmt.Sprint(v))
			if err != nil {
				return "", err
			}
			return RawJSON(b), nil
		},
		"squote": func(v any) string {
			return "'" + strings.ReplaceAll(fmt.Sprint(v), "'", "''") + "'"
		},
		"list": func(v ...any) []any {
			return v
		},
		"dict": dict,
		"toJson": func(v any) (RawJSON, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return RawJSON(b), nil
		},
		"toPrettyJson": func(v any) (RawJSON, error) {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return "", err
			}
			return RawJSON(b), nil
		},
		"toYaml":    serialize.MarshalYAML,
		"indent":    indent,
		"nindent":   nindent,
		"join":      join,
		"split":     strings.Split,
		"normalize": maps.NormalizeName,
		"required": func(msg string, value any) (any, error) {
			if maps.IsZero(value) {
				return nil, fmt.Errorf(msg)
			}
			return value, nil
		},
		"__typed__": typedJSON,
	}
}

func dict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments")
	}
	out := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T", kv[i])
		}
		out[key] = kv[i+1]
	}
	return out, nil
}

func indent(n int, s string) string {
	if s == "" {
		return ""
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] == "" {
			continue
		}
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func nindent(n int, s string) string {
	if s == "" {
		return ""
	}
	return "\n" + indent(n, s)
}

func join(sep string, v any) (string, error) {
	switch t := v.(type) {
	case []string:
		return strings.Join(t, sep), nil
	case []any:
		parts := make([]string, len(t))
		for i := range t {
			parts[i] = fmt.Sprint(t[i])
		}
		return strings.Join(parts, sep), nil
	default:
		return "", fmt.Errorf("join expects []string or []any, got %T", v)
	}
}
