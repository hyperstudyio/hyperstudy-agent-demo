package llama

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWaitReadyOrExitReturnsErrorWhenChildExitsFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503) // never becomes ready
	}))
	defer srv.Close()
	exited := make(chan error, 1)
	exited <- errors.New("exit status 1")
	err := WaitReadyOrExit(srv.URL, srv.Client(), 5*time.Second, exited)
	if err == nil || !strings.Contains(err.Error(), "llama-server exited before becoming ready") {
		t.Fatalf("want exit-detection error, got %v", err)
	}
}

func TestWaitReadyOrExitSucceedsBeforeChildExits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	exited := make(chan error, 1) // never sent — child stays alive
	if err := WaitReadyOrExit(srv.URL, srv.Client(), 5*time.Second, exited); err != nil {
		t.Fatalf("want success, got %v", err)
	}
}

func TestWaitReadyOrExitDoesNotHangOnClosedChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	exited := make(chan error)
	close(exited) // simulates a child that exited with no captured error
	start := time.Now()
	err := WaitReadyOrExit(srv.URL, srv.Client(), 5*time.Second, exited)
	if err == nil || !strings.Contains(err.Error(), "llama-server exited before becoming ready") {
		t.Fatalf("want exit-detection error, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("should have returned promptly on exit signal, took %s", time.Since(start))
	}
}

func TestBuildArgs(t *testing.T) {
	got := BuildArgs(LaunchOpts{HFRef: "org/repo:Q4_K_M", APIKey: "hsa_x"})
	want := []string{
		"-hf", "org/repo:Q4_K_M",
		"--api-key", "hsa_x",
		"--host", "0.0.0.0", "--port", "8080",
		"-np", "8", "-kvu", "-c", "32768",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildArgsOverrides(t *testing.T) {
	got := BuildArgs(LaunchOpts{HFRef: "r:Q", APIKey: "k", Port: 9000, Parallel: 4, Ctx: 65536})
	joined := strings.Join(got, " ")
	for _, frag := range []string{"--port 9000", "-np 4", "-c 65536"} {
		if !strings.Contains(joined, frag) {
			t.Fatalf("missing %q in %q", frag, joined)
		}
	}
}

func TestFindNotFoundMessageHasInstallHint(t *testing.T) {
	_, err := Find(func(string) (string, error) { return "", errors.New("nope") })
	if err == nil || !strings.Contains(err.Error(), "brew install llama.cpp") {
		t.Fatalf("want actionable install hint, got %v", err)
	}
}

func TestWaitReady(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(404)
			return
		}
		n++
		if n < 3 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	if err := WaitReady(srv.URL, srv.Client(), 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := WaitReady("http://127.0.0.1:1", &http.Client{Timeout: 200 * time.Millisecond}, 900*time.Millisecond); err == nil {
		t.Fatal("want timeout error against a dead port")
	}
}
