package llama

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestBuildArgsWithMTP(t *testing.T) {
	got := BuildArgs(LaunchOpts{HFRef: "org/repo:Q4_K_M", APIKey: "hsa_x", MTPFile: "/tmp/mtp.gguf"})
	// The four MTP args must appear together and in order.
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "--spec-type draft-mtp -md /tmp/mtp.gguf --spec-draft-n-max 3") {
		t.Fatalf("missing/wrong MTP args in %v", got)
	}
}

func TestBuildArgsWithoutMTP(t *testing.T) {
	got := BuildArgs(LaunchOpts{HFRef: "org/repo:Q4_K_M", APIKey: "hsa_x"})
	if strings.Contains(strings.Join(got, " "), "draft-mtp") {
		t.Fatalf("should not emit MTP args when MTPFile is empty: %v", got)
	}
}

func TestEnsureFileDownloadsThenCaches(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(200)
		w.Write([]byte("DRAFT-GGUF-BYTES"))
	}))
	defer srv.Close()
	dest := t.TempDir() + "/sub/mtp.gguf"

	p, err := EnsureFile(srv.URL, dest, srv.Client())
	if err != nil || p != dest {
		t.Fatalf("first download: p=%q err=%v", p, err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "DRAFT-GGUF-BYTES" {
		t.Fatalf("content mismatch: %q", b)
	}
	// Second call must NOT hit the network (file already present).
	if _, err := EnsureFile(srv.URL, dest, srv.Client()); err != nil {
		t.Fatalf("cached call: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected exactly 1 network hit, got %d", hits)
	}
}

func TestEnsureFileHTTPErrorNoPartialLeft(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()
	dest := t.TempDir() + "/mtp.gguf"
	if _, err := EnsureFile(srv.URL, dest, srv.Client()); err == nil {
		t.Fatal("want error on HTTP 404")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("a failed download must not leave a file at dest")
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
