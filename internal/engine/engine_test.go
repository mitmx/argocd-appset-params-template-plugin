package engine

import (
	"strings"
	"testing"
)

func TestResolveLegacyClusterOverrides(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{
		ApplicationSetName: "guestbook",
		Parameters: map[string]any{
			"appName": "guestbook",
			"name":    "dev-processing",
			"cluster": map[string]any{
				"name":           "dev-processing",
				"nameNormalized": "dev-processing",
			},
			"defaults": map[string]any{
				"domain": "aerxd.tech",
				"host":   "guestbook.{{ .name }}.{{ .domain }}",
			},
			"overridesByCluster": map[string]any{
				"dev-processing": map[string]any{"domain": "blabla.com"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	values := out[0]["resolvedValues"].(map[string]any)
	if got := values["host"]; got != "guestbook.dev-processing.blabla.com" {
		t.Fatalf("host = %v, want %q", got, "guestbook.dev-processing.blabla.com")
	}
}

func TestFanoutTargets(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{
		Parameters: map[string]any{
			"mode": "fanout",
			"name": "dev-p2p",
			"context": map[string]any{
				"cluster": map[string]any{"name": "dev-p2p"},
			},
			"targets": []any{
				map[string]any{
					"emit":     map[string]any{"appName": "guestbook"},
					"defaults": map[string]any{"host": "{{ .appName }}.{{ .name }}.example.com"},
				},
				map[string]any{
					"emit":     map[string]any{"appName": "billing"},
					"defaults": map[string]any{"host": "{{ .appName }}.{{ .name }}.example.com"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	for _, item := range out {
		appName := item["appName"].(string)
		values := item["resolvedValues"].(map[string]any)
		if got, want := values["host"], appName+".dev-example.example.com"; got != want {
			t.Fatalf("host = %v, want %q", got, want)
		}
	}
}

func TestSelectorOverrides(t *testing.T) {
	eng := New(4)
	out, err := eng.Execute(Request{
		Parameters: map[string]any{
			"mode": "fanout",
			"context": map[string]any{
				"cluster": map[string]any{"name": "prod-eu1"},
			},
			"selectorOverrides": []any{
				map[string]any{
					"path": "cluster.name",
					"overrides": map[string]any{
						"prod-eu1": map[string]any{"domain": "prod.example.com"},
					},
				},
			},
			"targets": []any{
				map[string]any{
					"defaults": map[string]any{
						"domain": "default.example.com",
						"host":   "svc.{{ .domain }}",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	values := out[0]["resolvedValues"].(map[string]any)
	if got := values["host"]; got != "svc.prod.example.com" {
		t.Fatalf("host = %v, want %q", got, "svc.prod.example.com")
	}
}

func TestCycleDetection(t *testing.T) {
	eng := New(2)
	_, err := eng.Execute(Request{
		Parameters: map[string]any{
			"defaults": map[string]any{
				"a": "{{ .b }}",
				"b": "{{ .a }}",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "did not stabilize") {
		t.Fatalf("unexpected error: %v", err)
	}
}
