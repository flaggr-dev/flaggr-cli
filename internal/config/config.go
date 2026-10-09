// Package config manages CLI configuration stored at ~/.flaggr/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds CLI credentials and defaults.
type Config struct {
	APIURL           string `json:"api_url"`
	EvalURL          string `json:"eval_url,omitempty"`
	APIToken         string `json:"api_token"`
	DefaultProjectID string `json:"default_project_id,omitempty"`
	DefaultServiceID string `json:"default_service_id,omitempty"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flaggr")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Path returns the config file path for display purposes.
func Path() string { return configPath() }

// Load reads the config file. Returns defaults if not found.
func Load() *Config {
	cfg := &Config{APIURL: "https://flaggr.dev"}

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	if cfg.APIURL == "" {
		cfg.APIURL = "https://flaggr.dev"
	}
	return cfg
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0600)
}

// Resolve returns the effective API URL and token, preferring env vars.
func Resolve(cfg *Config) (apiURL, apiToken string) {
	apiURL = cfg.APIURL
	if v := os.Getenv("FLAGGR_API_URL"); v != "" {
		apiURL = v
	}
	apiToken = cfg.APIToken
	if v := os.Getenv("FLAGGR_API_TOKEN"); v != "" {
		apiToken = v
	}
	return
}

// ResolveEvalURL returns the effective evaluation data plane URL.
// Defaults to https://api.flaggr.dev when API URL is https://flaggr.dev.
func ResolveEvalURL(cfg *Config) string {
	if v := os.Getenv("FLAGGR_EVAL_URL"); v != "" {
		return v
	}
	if cfg.EvalURL != "" {
		return cfg.EvalURL
	}
	apiURL, _ := Resolve(cfg)
	if apiURL == "https://flaggr.dev" {
		return "https://api.flaggr.dev"
	}
	return apiURL
}
