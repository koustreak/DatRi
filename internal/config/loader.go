package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Load reads a YAML config file from path and returns the parsed Config.
// After parsing, environment variable overrides are applied and the result
// is validated before being returned.
//
// If path is empty, Load returns an error — datri.yaml is always required
// in multi-resource mode.
func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("config: path is required")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: open %q: %w", path, err)
	}
	defer f.Close()

	cfg := &Config{}
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true) // reject unknown keys early — catches typos like `databse:`
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}

	applyDefaults(cfg)
	applyEnv(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyDefaults fills in zero-value fields with their sensible defaults.
// This runs before env var overrides so env vars always win.
func applyDefaults(cfg *Config) {
	// Log defaults
	if cfg.Global.Log.Level == "" {
		cfg.Global.Log.Level = "info"
	}
	if cfg.Global.Log.Format == "" {
		cfg.Global.Log.Format = "console"
	}

	// Per-resource defaults
	for i := range cfg.Resources {
		r := &cfg.Resources[i]

		// Server defaults
		if r.Server.Host == "" {
			r.Server.Host = "0.0.0.0"
		}
		if len(r.Server.Protocols) == 0 {
			r.Server.Protocols = []string{"rest"}
		}

		// Database pool/timeout defaults
		if r.Database != nil {
			p := &r.Database.Pool
			if p.MaxConns == 0 {
				p.MaxConns = 25
			}
			if p.MinConns == 0 {
				p.MinConns = 5
			}
			if p.MaxConnLifetime == 0 {
				p.MaxConnLifetime = 30 * time.Minute
			}
			if p.MaxConnIdleTime == 0 {
				p.MaxConnIdleTime = 5 * time.Minute
			}
			t := &r.Database.Timeouts
			if t.Connect == 0 {
				t.Connect = 10 * time.Second
			}
			if t.Query == 0 {
				t.Query = 30 * time.Second
			}
		}
	}
}

// applyEnv overlays environment variable values on top of the parsed config.
// Environment variables always take the highest precedence.
//
// Global overrides:
//
//	DATRI_LOG_LEVEL               → global.log.level
//	DATRI_LOG_FORMAT              → global.log.format
//	DATRI_AUTH_ENABLED            → global.auth.enabled
//	DATRI_AUTH_SECRET             → global.auth.secret
//
// Per-resource overrides (RESOURCE_NAME is the resource name uppercased, hyphens replaced by _):
//
//	DATRI_<NAME>_DATABASE_DSN     → resources[name].database.dsn
//	DATRI_<NAME>_AUTH_ENABLED     → resources[name].auth.enabled
//	DATRI_<NAME>_AUTH_SECRET      → resources[name].auth.secret
//	DATRI_<NAME>_ACCESS_KEY       → resources[name].filestore.access_key
//	DATRI_<NAME>_SECRET_KEY       → resources[name].filestore.secret_key
//	DATRI_<NAME>_SERVER_PORT      → resources[name].server.port
func applyEnv(cfg *Config) {
	// Global log
	if v := env("DATRI_LOG_LEVEL"); v != "" {
		cfg.Global.Log.Level = v
	}
	if v := env("DATRI_LOG_FORMAT"); v != "" {
		cfg.Global.Log.Format = v
	}

	// Global auth
	if v := env("DATRI_AUTH_ENABLED"); v != "" {
		enabled := strings.EqualFold(v, "true") || v == "1"
		if cfg.Global.Auth == nil {
			cfg.Global.Auth = &AuthConfig{}
		}
		cfg.Global.Auth.Enabled = enabled
	}
	if v := env("DATRI_AUTH_SECRET"); v != "" {
		if cfg.Global.Auth == nil {
			cfg.Global.Auth = &AuthConfig{}
		}
		cfg.Global.Auth.Secret = v
	}

	// Per-resource overrides
	for i := range cfg.Resources {
		r := &cfg.Resources[i]
		prefix := "DATRI_" + envKey(r.Name) + "_"

		// Server port
		if v := env(prefix + "SERVER_PORT"); v != "" {
			if p, err := strconv.Atoi(v); err == nil {
				r.Server.Port = p
			}
		}

		// Database DSN
		if r.Database != nil {
			if v := env(prefix + "DATABASE_DSN"); v != "" {
				r.Database.DSN = v
			}
		}

		// Filestore credentials
		if r.Filestore != nil {
			if v := env(prefix + "ACCESS_KEY"); v != "" {
				r.Filestore.AccessKey = v
			}
			if v := env(prefix + "SECRET_KEY"); v != "" {
				r.Filestore.SecretKey = v
			}
		}

		// Resource-specific auth
		if v := env(prefix + "AUTH_ENABLED"); v != "" {
			enabled := strings.EqualFold(v, "true") || v == "1"
			if r.Auth == nil {
				r.Auth = &AuthConfig{}
			}
			r.Auth.Enabled = enabled
		}
		if v := env(prefix + "AUTH_SECRET"); v != "" {
			if r.Auth == nil {
				r.Auth = &AuthConfig{}
			}
			r.Auth.Secret = v
		}
	}
}

// envKey converts a resource name into an env var key segment.
// "user-service" → "USER_SERVICE"
func envKey(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// env is a thin wrapper around os.Getenv that trims surrounding whitespace.
func env(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
