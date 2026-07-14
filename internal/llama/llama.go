package llama

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type LaunchOpts struct {
	HFRef, APIKey       string
	Port, Parallel, Ctx int
	NoThinking          bool
}

func (o LaunchOpts) withDefaults() LaunchOpts {
	if o.Port == 0 {
		o.Port = 8080
	}
	if o.Parallel == 0 {
		o.Parallel = 8
	}
	if o.Ctx == 0 {
		o.Ctx = 32768
	}
	return o
}

// BuildArgs assembles the exact llama-server invocation from the spec:
// continuous batching over -np slots sharing one unified KV pool (-kvu).
// KV-cache quantization is deliberately never emitted (degrades tool calling).
func BuildArgs(o LaunchOpts) []string {
	o = o.withDefaults()
	args := []string{
		"-hf", o.HFRef,
		"--api-key", o.APIKey,
		"--host", "0.0.0.0", "--port", strconv.Itoa(o.Port),
		"-np", strconv.Itoa(o.Parallel), "-kvu", "-c", strconv.Itoa(o.Ctx),
	}
	if o.NoThinking {
		args = append(args, "--chat-template-kwargs", `{"enable_thinking":false}`)
	}
	return args
}

func Find(look func(string) (string, error)) (string, error) {
	if p, err := look("llama-server"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf(`llama-server not found on PATH.
Install it:
  macOS:        brew install llama.cpp
  Linux x86_64: download a release from https://github.com/ggml-org/llama.cpp/releases
  DGX Spark:    build from source with -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=121 (see README)`)
}

func Start(bin string, o LaunchOpts) (*exec.Cmd, error) {
	cmd := exec.Command(bin, BuildArgs(o)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

// WaitReady polls /health until the server answers 200. First launch includes
// the model download, so callers pass a generous timeout.
func WaitReady(baseURL string, client *http.Client, timeout time.Duration) error {
	return WaitReadyOrExit(baseURL, client, timeout, nil)
}

// WaitReadyOrExit polls /health until the server answers 200, racing against
// exited: a channel the caller feeds with the child process's exec.Cmd.Wait()
// result. Without this race, a child that dies immediately (bad model ref,
// port already in use) leaves WaitReady polling a dead port for the full
// timeout instead of surfacing the real failure. Pass a nil channel to wait
// on readiness alone (that's what WaitReady does).
func WaitReadyOrExit(baseURL string, client *http.Client, timeout time.Duration, exited <-chan error) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("llama-server not ready after %s", timeout)
		}
		select {
		case exitErr := <-exited:
			if exitErr == nil {
				exitErr = fmt.Errorf("exit status 0")
			}
			return fmt.Errorf("llama-server exited before becoming ready — see its output above: %w", exitErr)
		case <-ticker.C:
		}
	}
}
