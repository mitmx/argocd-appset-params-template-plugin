package serialize

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mitmx/argocd-values-pipeline-plugin/internal/maps"
)

func MarshalJSONPretty(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func MarshalYAML(v any) (string, error) {
	var b strings.Builder
	if err := writeYAML(&b, v, 0); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeYAML(b *strings.Builder, v any, indent int) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("null\n")
		return nil
	case map[string]any:
		if len(t) == 0 {
			b.WriteString("{}\n")
			return nil
		}
		for _, key := range maps.SortedKeys(t) {
			writeIndent(b, indent)
			b.WriteString(quoteKey(key))
			value := t[key]
			if isScalar(value) {
				b.WriteString(": ")
				if err := writeScalar(b, value); err != nil {
					return err
				}
				b.WriteByte('\n')
				continue
			}
			b.WriteString(":\n")
			if err := writeYAML(b, value, indent+2); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if len(t) == 0 {
			b.WriteString("[]\n")
			return nil
		}
		for _, item := range t {
			writeIndent(b, indent)
			if isScalar(item) {
				b.WriteString("- ")
				if err := writeScalar(b, item); err != nil {
					return err
				}
				b.WriteByte('\n')
				continue
			}
			b.WriteString("-\n")
			if err := writeYAML(b, item, indent+2); err != nil {
				return err
			}
		}
		return nil
	default:
		if isScalar(v) {
			if err := writeScalar(b, v); err != nil {
				return err
			}
			b.WriteByte('\n')
			return nil
		}
		return fmt.Errorf("unsupported yaml value type %T", v)
	}
}

func isScalar(v any) bool {
	switch v.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

func writeScalar(b *strings.Builder, v any) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case string:
		writeString(b, t)
	case bool:
		b.WriteString(strconv.FormatBool(t))
	case int:
		b.WriteString(strconv.Itoa(t))
	case int8, int16, int32, int64:
		b.WriteString(fmt.Sprintf("%d", t))
	case uint, uint8, uint16, uint32, uint64:
		b.WriteString(fmt.Sprintf("%d", t))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(t), 'f', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(t, 'f', -1, 64))
	default:
		return fmt.Errorf("unsupported scalar type %T", v)
	}
	return nil
}

func writeString(b *strings.Builder, s string) {
	if maps.IsSafeBareString(s) {
		b.WriteString(s)
		return
	}
	b.WriteString(strconv.Quote(s))
}

func writeIndent(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
}

func quoteKey(s string) string {
	if maps.IsSafeBareString(s) {
		return s
	}
	return strconv.Quote(s)
}
