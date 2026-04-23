package engine

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveContractWithParamsAndSelectors(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{Parameters: map[string]any{
		"context": map[string]any{"cluster": map[string]any{"name": "dev-bar", "nameNormalized": "dev-bar"}},
		"params":  map[string]any{"appName": "guestbook"},
		"defaults": map[string]any{
			"domain": "example.com",
			"host":   "{{ .params.appName }}.{{ .context.cluster.name }}.{{ .result.domain }}",
		},
		"selectorOverrides": []any{
			map[string]any{
				"path": "context.cluster.name",
				"overrides": map[string]any{
					"dev-bar": map[string]any{"domain": "blabla.com"},
				},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	result := out[0]["result"].(map[string]any)
	if got := result["host"]; got != "guestbook.dev-bar.blabla.com" {
		t.Fatalf("host = %v", got)
	}
	if got := out[0]["params"].(map[string]any)["appName"]; got != "guestbook" {
		t.Fatalf("params.appName = %v", got)
	}
}

func TestTargetsFanOutWithoutMode(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{Parameters: map[string]any{
		"context":  map[string]any{"cluster": map[string]any{"name": "dev-foo"}},
		"defaults": map[string]any{"host": "{{ .params.appName }}.{{ .context.cluster.name }}.example.com"},
		"targets": []any{
			map[string]any{"params": map[string]any{"appName": "guestbook"}},
			map[string]any{"params": map[string]any{"appName": "billing"}},
		},
	}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	for _, item := range out {
		appName := item["params"].(map[string]any)["appName"].(string)
		result := item["result"].(map[string]any)
		if got, want := result["host"], appName+".dev-foo.example.com"; got != want {
			t.Fatalf("host = %v, want %q", got, want)
		}
	}
}

func TestSelectorDefaultApplied(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{Parameters: map[string]any{
		"context":  map[string]any{"cluster": map[string]any{"name": "prod-eu1"}},
		"defaults": map[string]any{"replicas": 1},
		"selectorOverrides": []any{
			map[string]any{
				"path": "context.cluster.name",
				"overrides": map[string]any{
					"dev-foo": map[string]any{"replicas": 2},
				},
				"default": map[string]any{"replicas": 3},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out[0]["result"].(map[string]any)["replicas"]; got != 3 {
		t.Fatalf("replicas = %v, want 3", got)
	}
}

func TestResultSupportsMultiSourceShape(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{Parameters: map[string]any{
		"context": map[string]any{"cluster": map[string]any{"name": "dev-foo"}},
		"defaults": map[string]any{
			"operator": map[string]any{"resources": map[string]any{}},
			"cluster":  map[string]any{"cephClusterSpec": map[string]any{"mon": map[string]any{"count": 3}}},
		},
		"selectorOverrides": []any{
			map[string]any{
				"path": "context.cluster.name",
				"overrides": map[string]any{
					"dev-foo": map[string]any{
						"cluster": map[string]any{
							"cephClusterSpec": map[string]any{"mon": map[string]any{"count": 1}},
						},
					},
				},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := out[0]["result"].(map[string]any)
	operator := result["operator"].(map[string]any)
	cluster := result["cluster"].(map[string]any)
	if _, ok := operator["resources"].(map[string]any); !ok {
		t.Fatalf("operator.resources missing or wrong type: %#v", operator)
	}
	got := cluster["cephClusterSpec"].(map[string]any)["mon"].(map[string]any)["count"]
	if got != 1 {
		t.Fatalf("cluster.cephClusterSpec.mon.count = %v, want 1", got)
	}
}

func TestRejectUnknownTopLevelField(t *testing.T) {
	eng := New(4)
	_, err := eng.Execute(Request{Parameters: map[string]any{
		"defaults": map[string]any{"domain": "example.com"},
		"name":     "dev-foo",
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown field \"name\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectInvalidSelectorPath(t *testing.T) {
	eng := New(4)
	_, err := eng.Execute(Request{Parameters: map[string]any{
		"selectorOverrides": []any{
			map[string]any{
				"path":      "cluster.name",
				"overrides": map[string]any{"dev": map[string]any{"domain": "dev.example.com"}},
			},
		},
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must start with \"context.\" or \"params.\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectTargetDefaults(t *testing.T) {
	eng := New(4)
	_, err := eng.Execute(Request{Parameters: map[string]any{
		"targets": []any{map[string]any{"defaults": map[string]any{"domain": "example.com"}}},
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown field \"defaults\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCycleDetection(t *testing.T) {
	eng := New(2)
	_, err := eng.Execute(Request{Parameters: map[string]any{
		"defaults": map[string]any{"a": "{{ .result.b }}", "b": "{{ .result.a }}"},
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unresolved template remains") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTargetContextAvailableDuringRender(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{Parameters: map[string]any{
		"defaults": map[string]any{"host": "{{ .target.params.appName }}.{{ .target.context.env }}.example.com"},
		"targets":  []any{map[string]any{"context": map[string]any{"env": "prod"}, "params": map[string]any{"appName": "guestbook"}}},
	}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := out[0]["result"].(map[string]any)
	if result["host"] != "guestbook.prod.example.com" {
		t.Fatalf("host = %v", result["host"])
	}
	if !reflect.DeepEqual(out[0]["params"], map[string]any{"appName": "guestbook"}) {
		t.Fatalf("params = %#v", out[0]["params"])
	}
}
