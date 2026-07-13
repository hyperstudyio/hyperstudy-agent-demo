package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDir(t *testing.T) {
	dir := DefaultDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".hyperstudy")
	if dir != expected {
		t.Errorf("DefaultDir() = %q, want %q", dir, expected)
	}
}

func TestLoadCreatesMissing(t *testing.T) {
	tmpdir := t.TempDir()
	c, err := Load(tmpdir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c == nil {
		t.Fatal("Load() returned nil")
	}
	if c.Key == "" {
		t.Error("Load() did not auto-generate Key")
	}
}

func TestEnsureKeyGenerates(t *testing.T) {
	c := &Config{}
	if err := c.EnsureKey(); err != nil {
		t.Fatalf("EnsureKey() error = %v", err)
	}
	if len(c.Key) != 68 { // "hsa_" (4) + 64 hex chars (32 bytes * 2)
		t.Errorf("EnsureKey() Key length = %d, want 68", len(c.Key))
	}
	if c.Key[:4] != "hsa_" {
		t.Errorf("EnsureKey() Key prefix = %q, want %q", c.Key[:4], "hsa_")
	}
}

func TestEnsureKeyIdempotent(t *testing.T) {
	c := &Config{}
	c.EnsureKey()
	key1 := c.Key
	c.EnsureKey()
	if c.Key != key1 {
		t.Error("EnsureKey() is not idempotent")
	}
}

func TestSaveLoad(t *testing.T) {
	tmpdir := t.TempDir()
	c := &Config{Key: "hsa_deadbeef"}
	if err := c.Save(tmpdir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	c2, err := Load(tmpdir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c2.Key != c.Key {
		t.Errorf("Load() Key = %q, want %q", c2.Key, c.Key)
	}
}
