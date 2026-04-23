package maps

import "testing"

func TestDeepMerge(t *testing.T) {
	got := DeepMerge(
		map[string]any{"a": map[string]any{"b": 1, "c": []any{"x"}}},
		map[string]any{"a": map[string]any{"b": 2, "d": true}},
	)
	inner := got["a"].(map[string]any)
	if inner["b"] != 2 {
		t.Fatalf("b = %v, want 2", inner["b"])
	}
	if inner["d"] != true {
		t.Fatalf("d = %v, want true", inner["d"])
	}
}

func TestLookupPath(t *testing.T) {
	v, ok, err := LookupPath(map[string]any{
		"cluster": map[string]any{"name": "dev"},
		"list":    []any{map[string]any{"x": 1}},
	}, "list.0.x")
	if err != nil {
		t.Fatalf("LookupPath() error = %v", err)
	}
	if !ok || v != 1 {
		t.Fatalf("LookupPath() = (%v, %v), want (1, true)", v, ok)
	}
}
