// internal/verify/verify_test.go
package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

// toolCallStub returns a server whose /chat/completions always answers with a
// single "respond" tool call carrying the given raw arguments JSON.
func toolCallStub(t *testing.T, argsJSON string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
			"content": "",
			"tool_calls": []any{map[string]any{
				"id": "c1", "type": "function",
				"function": map[string]any{"name": "respond", "arguments": argsJSON},
			}},
		}, "finish_reason": "tool_calls"}}})
	}))
}

func TestToolCallArgsMissingValueFailsSchema(t *testing.T) {
	srv := toolCallStub(t, `{"answer":2}`)
	defer srv.Close()
	r := CheckToolCall(srv.URL+"/v1", key, srv.Client())
	if r.Pass {
		t.Fatal("arguments missing the required 'value' field must FAIL the schema check")
	}
	if !strings.Contains(r.Detail, "value") {
		t.Fatalf("detail should name the missing 'value' field: %s", r.Detail)
	}
}

func TestToolCallArgsWrongTypeFailsSchema(t *testing.T) {
	srv := toolCallStub(t, `{"value":"two"}`)
	defer srv.Close()
	r := CheckToolCall(srv.URL+"/v1", key, srv.Client())
	if r.Pass {
		t.Fatal("arguments with wrong-typed 'value' must FAIL the schema check")
	}
	if !strings.Contains(r.Detail, "value") {
		t.Fatalf("detail should name the wrong-type 'value' field: %s", r.Detail)
	}
}

// makeDurs builds n durations sorted so that the highest 'tail' of them equal
// hi and the rest equal lo, exercising a specific p95 value once sorted.
func makeDurs(n, tail int, lo, hi time.Duration) []time.Duration {
	durs := make([]time.Duration, n)
	for i := 0; i < n-tail; i++ {
		durs[i] = lo
	}
	for i := n - tail; i < n; i++ {
		durs[i] = hi
	}
	return durs
}

func TestConcurrencyResultAllFastPasses(t *testing.T) {
	durs := makeDurs(4, 0, 2*time.Second, 2*time.Second)
	r := concurrencyResult("concurrency (4 parallel)", durs)
	if !r.Pass {
		t.Fatalf("fast durations should pass: %s", r.Detail)
	}
	if strings.Contains(r.Detail, "raise the credential's timeoutMs") {
		t.Fatalf("fast durations should not carry the warn text: %s", r.Detail)
	}
}

func TestConcurrencyResultWarnsAboveSixtySeconds(t *testing.T) {
	// n=100, p95 index = (100*95)/100 = 95 -> need the top 5 of 100 to be the
	// p95 value once sorted ascending.
	durs := makeDurs(100, 5, 2*time.Second, 70*time.Second)
	r := concurrencyResult("concurrency (100 parallel)", durs)
	if !r.Pass {
		t.Fatalf("p95=70s should still pass: %s", r.Detail)
	}
	if !strings.Contains(r.Detail, "raise the credential's timeoutMs") {
		t.Fatalf("detail should contain the timeoutMs warning: %s", r.Detail)
	}
}

func TestConcurrencyResultFailsAbove300Seconds(t *testing.T) {
	durs := makeDurs(100, 5, 2*time.Second, 301*time.Second)
	r := concurrencyResult("concurrency (100 parallel)", durs)
	if r.Pass {
		t.Fatal("p95=301s should fail")
	}
	if !strings.Contains(r.Detail, "300s") {
		t.Fatalf("detail should mention the 300s platform maximum: %s", r.Detail)
	}
}

func TestConcurrencyResultSingleDurationNoPanic(t *testing.T) {
	r := concurrencyResult("concurrency (1 parallel)", []time.Duration{2 * time.Second})
	if !r.Pass {
		t.Fatalf("single fast duration should pass: %s", r.Detail)
	}
}
