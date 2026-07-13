package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey string `json:"apiKey"`
	Model  string `json:"model,omitempty"`
	Port   int    `json:"port,omitempty"`
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hyperstudy-agent"
	}
	return filepath.Join(home, ".hyperstudy-agent")
}

func Load(dir string) (*Config, error) {
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), b, 0o600)
}

// EnsureKey generates an hsa_-prefixed key if none is set. The key is what the
// researcher pastes into HyperStudy Settings AND what llama-server enforces.
func (c *Config) EnsureKey() bool {
	if c.APIKey != "" {
		return false
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	c.APIKey = "hsa_" + hex.EncodeToString(buf)
	return true
}
