package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Key string `json:"key"`
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hyperstudy"
	}
	return filepath.Join(home, ".hyperstudy")
}

func Load(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	c := &Config{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, c); err != nil {
			return nil, err
		}
	}

	if c.Key == "" {
		if err := c.EnsureKey(); err != nil {
			return nil, err
		}
		if err := c.Save(dir); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Config) Save(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

func (c *Config) EnsureKey() error {
	if c.Key != "" {
		return nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	c.Key = "hsa_" + hex.EncodeToString(buf)
	return nil
}
