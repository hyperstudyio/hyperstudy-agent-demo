package config

import (
	"regexp"
	"testing"
)

func TestEnsureKeyGeneratesOnce(t *testing.T) {
	c := &Config{}
	if gen := c.EnsureKey(); !gen {
		t.Fatal("expected key generation on empty config")
	}
	if !regexp.MustCompile(`^hsa_[0-9a-f]{64}$`).MatchString(c.APIKey) {
		t.Fatalf("bad key format: %q", c.APIKey)
	}
	first := c.APIKey
	if gen := c.EnsureKey(); gen {
		t.Fatal("must not regenerate an existing key")
	}
	if c.APIKey != first {
		t.Fatal("key changed on second EnsureKey")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := &Config{APIKey: "hsa_abc", Model: "m", Port: 8080}
	if err := c.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if *got != *c {
		t.Fatalf("round trip mismatch: %+v != %+v", got, c)
	}
}

func TestLoadMissingIsZero(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil || got.APIKey != "" {
		t.Fatalf("want zero config, got %+v err %v", got, err)
	}
}
