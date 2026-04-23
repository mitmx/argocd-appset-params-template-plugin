package maps

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func DeepCopy(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = DeepCopy(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = DeepCopy(t[i])
		}
		return out
	default:
		return t
	}
}

func DeepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	out := DeepCopy(dst).(map[string]any)
	for k, srcValue := range src {
		existing, exists := out[k]
		if !exists {
			out[k] = DeepCopy(srcValue)
			continue
		}
		srcMap, srcIsMap := srcValue.(map[string]any)
		dstMap, dstIsMap := existing.(map[string]any)
		if srcIsMap && dstIsMap {
			out[k] = DeepMerge(dstMap, srcMap)
			continue
		}
		out[k] = DeepCopy(srcValue)
	}
	return out
}

func MergeAll(items ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, item := range items {
		out = DeepMerge(out, item)
	}
	return out
}

func CloneMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return DeepCopy(v).(map[string]any)
}

func SortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func LookupPath(root any, path string) (any, bool, error) {
	if path == "" {
		return root, true, nil
	}
	segments := strings.Split(path, ".")
	current := root
	for _, segment := range segments {
		switch typed := current.(type) {
		case map[string]any:
			v, ok := typed[segment]
			if !ok {
				return nil, false, nil
			}
			current = v
		case []any:
			idx, err := strconv.Atoi(segment)
			if err != nil {
				return nil, false, fmt.Errorf("path segment %q is not a slice index", segment)
			}
			if idx < 0 || idx >= len(typed) {
				return nil, false, nil
			}
			current = typed[idx]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func ToStringKey(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case fmt.Stringer:
		return t.String(), true
	case int:
		return strconv.Itoa(t), true
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", t), true
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", t), true
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return "", false
	}
}

func NormalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "default"
	}
	return result
}

func IsZero(v any) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).IsZero()
}

func IsSafeBareString(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	switch lower {
	case "null", "true", "false", "yes", "no", "on", "off", "~":
		return false
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return false
	}
	for i, r := range s {
		if unicode.IsSpace(r) {
			return false
		}
		switch r {
		case ':', '#', '{', '}', '[', ']', ',', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
			return false
		case '-':
			if i == 0 && len(s) > 1 {
				return false
			}
		}
	}
	return true
}
