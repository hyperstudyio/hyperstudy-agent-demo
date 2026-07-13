// internal/verify/verify.go
package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Result struct {
	Name   string
	Pass   bool
	Detail string
}

// respondTool mirrors the platform's llmDecide respond-style tool so a pass
// here predicts a pass in a real experiment turn.
var respondTool = map[string]any{
	"type": "function",
	"function": map[string]any{
		"name":        "respond",
		"description": "Submit your response",
		"parameters": map[string]any{
			"type":       "object",
			"properties": map[string]any{"value": map[string]any{"type": "number"}},
			"required":   []string{"value"},
		},
	},
}

func completionReq(baseURL, apiKey string, withTool bool) (*http.Request, error) {
	body := map[string]any{
		"model":      "default",
		"max_tokens": 64,
		"messages":   []any{map[string]any{"role": "user", "content": "Use the respond tool to submit the number 2."}},
	}
	if withTool {
		body["tools"] = []any{respondTool}
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func CheckModels(baseURL, apiKey string, client *http.Client) Result {
	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return Result{"reachable (/models)", false, fmt.Sprintf("endpoint unreachable: %v", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 401 && resp.StatusCode != 403 {
		return Result{"reachable (/models)", false, fmt.Sprintf("GET /models returned %d — not an OpenAI-compatible baseUrl?", resp.StatusCode)}
	}
	return Result{"reachable (/models)", true, "note: /models is served without auth on llama-server; auth is checked next"}
}

func CheckAuthEnforced(baseURL, apiKey string, client *http.Client) Result {
	req, err := completionReq(baseURL, "definitely-wrong-key", false)
	if err != nil {
		return Result{"auth enforced", false, err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{"auth enforced", false, err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return Result{"auth enforced", true, "wrong key rejected with " + resp.Status}
	}
	return Result{"auth enforced", false, fmt.Sprintf("a WRONG key was accepted (%d). Your server does not validate Authorization — anyone who finds the URL can use your model. llama-server: pass --api-key; Ollama has no auth (use llama-server or a reverse proxy).", resp.StatusCode)}
}

func CheckToolCall(baseURL, apiKey string, client *http.Client) Result {
	req, err := completionReq(baseURL, apiKey, true)
	if err != nil {
		return Result{"tool calling", false, err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{"tool calling", false, err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Result{"tool calling", false, fmt.Sprintf("chat/completions returned %d", resp.StatusCode)}
	}
	var out struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Name      string
						Arguments string
					}
				} `json:"tool_calls"`
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Choices) == 0 {
		return Result{"tool calling", false, "response is not OpenAI-shaped"}
	}
	tcs := out.Choices[0].Message.ToolCalls
	if len(tcs) == 0 {
		return Result{"tool calling", false, "no tool_calls in response — HyperStudy agents REQUIRE function calling. The model may not support tools or its chat template is broken; try a Qwen3 GGUF from unsloth."}
	}
	var args map[string]any
	if json.Unmarshal([]byte(tcs[0].Function.Arguments), &args) != nil {
		return Result{"tool calling", false, "tool_calls arguments are not valid JSON"}
	}
	return Result{"tool calling", true, fmt.Sprintf("model called %s(%s)", tcs[0].Function.Name, tcs[0].Function.Arguments)}
}

func CheckConcurrency(baseURL, apiKey string, n int, client *http.Client) Result {
	if n <= 0 {
		n = 8
	}
	durs := make([]time.Duration, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := time.Now()
			req, err := completionReq(baseURL, apiKey, true)
			if err == nil {
				var resp *http.Response
				resp, err = client.Do(req)
				if resp != nil {
					resp.Body.Close()
					if err == nil && resp.StatusCode != 200 {
						err = fmt.Errorf("status %d", resp.StatusCode)
					}
				}
			}
			durs[i], errs[i] = time.Since(start), err
		}(i)
	}
	wg.Wait()
	name := fmt.Sprintf("concurrency (%d parallel)", n)
	for _, e := range errs {
		if e != nil {
			return Result{name, false, fmt.Sprintf("request failed under load: %v", e)}
		}
	}
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	p50, p95 := durs[n/2], durs[(n*95)/100]
	detail := fmt.Sprintf("p50=%s p95=%s", p50.Round(time.Millisecond), p95.Round(time.Millisecond))
	if p95 > 300*time.Second {
		return Result{name, false, detail + " — exceeds the platform's 300s maximum timeout"}
	}
	if p95 > 60*time.Second {
		detail += " — above the 60s default; raise the credential's timeoutMs in HyperStudy Settings"
	}
	return Result{name, true, detail}
}

func Run(baseURL, apiKey string, concurrency int, client *http.Client) []Result {
	return []Result{
		CheckModels(baseURL, apiKey, client),
		CheckAuthEnforced(baseURL, apiKey, client),
		CheckToolCall(baseURL, apiKey, client),
		CheckConcurrency(baseURL, apiKey, concurrency, client),
	}
}
