package render

import (
	"reflect"
	"testing"
)

func TestRenderListAndMaps(t *testing.T) {
	r := New(4)
	got, err := r.Resolve(map[string]any{
		"domain": "example.com",
		"hosts":  []any{"{{ .values.domain }}", "api.{{ .values.domain }}"},
	}, func(current map[string]any) map[string]any {
		return map[string]any{"values": current}
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	hosts := got["hosts"].([]any)
	if hosts[0] != "example.com" || hosts[1] != "api.example.com" {
		t.Fatalf("hosts = %#v", hosts)
	}
}

func TestPureExpressionsKeepTypes(t *testing.T) {
	r := New(4)
	got, err := r.Resolve(map[string]any{
		"replicas":    "{{ 3 }}",
		"enabled":     "{{ true }}",
		"ratio":       "{{ 1.5 }}",
		"nothing":     "{{ toJson nil }}",
		"stringy":     "{{ quote \"1\" }}",
		"stringValue": "01",
		"copied":      "{{ .values.stringValue }}",
		"ports":       "{{ list 80 443 }}",
		"labels":      "{{ dict \"tier\" \"frontend\" \"track\" \"stable\" }}",
	}, func(current map[string]any) map[string]any {
		return map[string]any{"values": current}
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got["replicas"] != 3 {
		t.Fatalf("replicas = %#v, want int(3)", got["replicas"])
	}
	if got["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", got["enabled"])
	}
	if got["ratio"] != 1.5 {
		t.Fatalf("ratio = %#v, want 1.5", got["ratio"])
	}
	if got["nothing"] != nil {
		t.Fatalf("nothing = %#v, want nil", got["nothing"])
	}
	if got["stringy"] != "1" {
		t.Fatalf("stringy = %#v, want \"1\"", got["stringy"])
	}
	if got["copied"] != "01" {
		t.Fatalf("copied = %#v, want \"01\"", got["copied"])
	}
	if !reflect.DeepEqual(got["ports"], []any{80, 443}) {
		t.Fatalf("ports = %#v", got["ports"])
	}
	if !reflect.DeepEqual(got["labels"], map[string]any{"tier": "frontend", "track": "stable"}) {
		t.Fatalf("labels = %#v", got["labels"])
	}
}

func TestMixedTemplatesAlwaysStayStrings(t *testing.T) {
	r := New(4)
	got, err := r.Resolve(map[string]any{
		"port": "80",
		"host": "svc-{{ .values.port }}",
	}, func(current map[string]any) map[string]any {
		return map[string]any{"values": current}
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got["host"] != "svc-80" {
		t.Fatalf("host = %#v, want \"svc-80\"", got["host"])
	}
}

func TestPureExpressionWithToJsonStaysTyped(t *testing.T) {
	r := New(4)
	got, err := r.Resolve(map[string]any{
		"ports": "{{ toJson (list 80 443) }}",
	}, func(current map[string]any) map[string]any {
		return map[string]any{"values": current}
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(got["ports"], []any{80, 443}) {
		t.Fatalf("ports = %#v", got["ports"])
	}
}
