package render

import "testing"

func TestRenderListAndMaps(t *testing.T) {
	r := New(4)
	got, err := r.Resolve(map[string]any{
		"domain": "example.com",
		"hosts":  []any{"{{ .domain }}", "api.{{ .domain }}"},
	}, func(current map[string]any) map[string]any {
		return current
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	hosts := got["hosts"].([]any)
	if hosts[0] != "example.com" || hosts[1] != "api.example.com" {
		t.Fatalf("hosts = %#v", hosts)
	}
}
