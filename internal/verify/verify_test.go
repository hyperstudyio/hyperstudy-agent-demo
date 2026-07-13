// internal/verify/verify_test.go
package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const key = "hsa_good"

// healthyStub implements /v1/models + /v1/chat/completions with auth + tool calls.
func healthyStub(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/models"):
			json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "m"}}})
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			if r.Header.Get("Authorization") != "Bearer "+key {
				w.WriteHeader(401)
				return
			}
			var req struct {
				Tools []struct {
					Function struct{ Name string }
				}
			}
			json.NewDecoder(r.Body).Decode(&req)
			msg := map[string]any{"content": "hi"}
			finish := "stop"
			if len(req.Tools) > 0 {
				msg = map[string]any{"content": "", "tool_calls": []any{map[string]any{
					"id": "c1", "type": "function",
					"function": map[string]any{"name": req.Tools[0].Function.Name, "arguments": `{"value":2}`},
				}}}
				finish = "tool_calls"
			}
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": msg, "finish_reason": finish}}})
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestHealthyEndpointPassesAll(t *testing.T) {
	srv := healthyStub(t)
	defer srv.Close()
	results := Run(srv.URL+"/v1", key, 4, srv.Client())
	for _, r := range results {
		if !r.Pass {
			t.Errorf("%s failed: %s", r.Name, r.Detail)
		}
	}
	if len(results) != 4 {
		t.Fatalf("want 4 checks, got %d", len(results))
	}
}

func TestAuthNotEnforcedFails(t *testing.T) {
	// accepts ANY key — the Ollama trap
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "x"}}}, "data": []any{}})
	}))
	defer srv.Close()
	r := CheckAuthEnforced(srv.URL+"/v1", key, srv.Client())
	if r.Pass {
		t.Fatal("endpoint accepting a wrong key must FAIL the auth check")
	}
}

func TestTextOnlyModelFailsToolCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "I would respond with 2"}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	r := CheckToolCall(srv.URL+"/v1", key, srv.Client())
	if r.Pass {
		t.Fatal("text-only response must FAIL the tool-call check")
	}
	if !strings.Contains(r.Detail, "tool_calls") {
		t.Fatalf("detail should explain the tool_calls absence: %s", r.Detail)
	}
}

func TestUnreachableFailsModels(t *testing.T) {
	r := CheckModels("http://127.0.0.1:1/v1", key, &http.Client{})
	if r.Pass {
		t.Fatal("dead endpoint must fail")
	}
}
