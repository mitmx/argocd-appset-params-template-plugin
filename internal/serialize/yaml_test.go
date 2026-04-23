package serialize

import (
	"strings"
	"testing"
)

func TestMarshalYAML(t *testing.T) {
	out, err := MarshalYAML(map[string]any{
		"name":     "guestbook",
		"replicas": 2,
		"enabled":  true,
		"labels": map[string]any{
			"tier": "frontend",
		},
		"ports": []any{80, 443},
	})
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}
	for _, expected := range []string{
		"enabled: true\n",
		"labels:\n  tier: frontend\n",
		"name: guestbook\n",
		"ports:\n  - 80\n  - 443\n",
		"replicas: 2\n",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("yaml output missing %q\n%s", expected, out)
		}
	}
}
