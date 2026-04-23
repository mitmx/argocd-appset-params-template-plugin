package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/config"
	"github.com/mitmx/argocd-appset-params-template-plugin/internal/engine"
)

type Server struct {
	cfg    config.Config
	engine engine.Engine
}

type executeRequest struct {
	ApplicationSetName string `json:"applicationSetName"`
	Input              struct {
		Parameters map[string]any `json:"parameters"`
		Values     map[string]any `json:"values"`
	} `json:"input"`
}

type executeResponse struct {
	Output struct {
		Parameters []map[string]any `json:"parameters"`
	} `json:"output"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(cfg config.Config, eng engine.Engine) *Server {
	return &Server{cfg: cfg, engine: eng}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/getparams.execute":
		s.handleExecute(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r.Header.Get("Authorization")) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "forbidden"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	defer r.Body.Close()

	var req executeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		if isTooLarge(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request body exceeds configured limit"})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "decode request: " + err.Error()})
		return
	}
	if err := ensureSingleJSONDocument(decoder); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if req.Input.Parameters == nil {
		req.Input.Parameters = map[string]any{}
	}

	params, err := s.engine.Execute(engine.Request{Parameters: normalizeNumbers(req.Input.Parameters).(map[string]any)})
	if err != nil {
		log.Printf("execute error: %v", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	var resp executeResponse
	resp.Output.Parameters = params
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) authorized(header string) bool {
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	candidate := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(s.cfg.Token)) == 1
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ensureSingleJSONDocument(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("decode request: unexpected trailing JSON data")
	} else if !errors.Is(err, io.EOF) {
		return errors.New("decode request: invalid trailing JSON data")
	}
	return nil
}

func isTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func normalizeNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = normalizeNumbers(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = normalizeNumbers(t[i])
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
