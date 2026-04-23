package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RawJSON marks a string as already-serialized JSON produced intentionally from
// inside a template function. It lets pure-expression rendering preserve types
// without guessing whether an arbitrary string should be decoded as JSON.
type RawJSON string

func decodeJSONValue(rendered string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(rendered))
	decoder.UseNumber()

	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode typed json: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("decode typed json: unexpected trailing data")
	}
	return normalizeJSONNumbers(out), nil
}

func typedJSON(v any) (string, error) {
	switch t := v.(type) {
	case RawJSON:
		if !json.Valid([]byte(t)) {
			return "", fmt.Errorf("invalid raw json value")
		}
		return string(t), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func normalizeJSONNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for key, value := range t {
			out[key] = normalizeJSONNumbers(value)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = normalizeJSONNumbers(t[i])
		}
		return out
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	default:
		return v
	}
}
