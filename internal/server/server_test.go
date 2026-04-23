package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/config"
	"github.com/mitmx/argocd-appset-params-template-plugin/internal/engine"
)

func TestExecuteSuccess(t *testing.T) {
	srv := New(config.Config{Token: "secret", MaxBodyBytes: 1024}, engine.New(4))

	body := `{
        "applicationSetName":"guestbook",
        "input":{
            "parameters":{
                "context":{"cluster":{"name":"dev-foo"}},
                "params":{"appName":"guestbook"},
                "defaults":{"host":"{{ .params.appName }}.{{ .context.cluster.name }}.example.com"}
            }
        }
    }`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	output := resp["output"].(map[string]any)
	parameters := output["parameters"].([]any)
	first := parameters[0].(map[string]any)
	result := first["result"].(map[string]any)
	if got := result["host"]; got != "guestbook.dev-foo.example.com" {
		t.Fatalf("host = %v, want guestbook.dev-foo.example.com", got)
	}
}

func TestExecuteForbiddenWithoutToken(t *testing.T) {
	srv := New(config.Config{Token: "secret", MaxBodyBytes: 1024}, engine.New(4))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(`{"input":{"parameters":{}}}`))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestExecuteRejectsLargeBody(t *testing.T) {
	srv := New(config.Config{Token: "secret", MaxBodyBytes: 16}, engine.New(4))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(`{"input":{"parameters":{}}}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}
