package config

import (
	"fmt"
	"strings"
)

// ─── Allowed value sets ───────────────────────────────────────────────────────

var validResourceTypes = map[string]bool{
	"database":  true,
	"filestore": true,
}

var validProtocols = map[string]bool{
	"rest":      true,
	"graphql":   true,
	"grpc":      true,
	"websocket": true,
}

var validDBDrivers = map[string]bool{
	"postgres": true,
	"mysql":    true,
}

var validAuthTypes = map[string]bool{
	"jwt":    true,
	"apikey": true,
}

var validFilestoreProviders = map[string]bool{
	"minio": true,
	"s3":    true,
}

var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

var validLogFormats = map[string]bool{
	"json":    true,
	"console": true,
}

// ─── Validate ─────────────────────────────────────────────────────────────────

// Validate checks the entire Config for correctness:
//   - global settings are valid
//   - at least one resource is declared
//   - all resource names are unique
//   - all resource ports are unique and in a valid range
//   - each resource's type-specific block is present and valid
//   - auth and CORS blocks (global and per-resource) are valid
func (c *Config) Validate() error {
	if err := c.validateGlobal(); err != nil {
		return err
	}

	if len(c.Resources) == 0 {
		return fmt.Errorf("config: no resources declared — add at least one entry under 'resources:'")
	}

	names := make(map[string]bool, len(c.Resources))
	ports := make(map[int]string, len(c.Resources)) // port → resource name

	for i := range c.Resources {
		r := &c.Resources[i]

		if err := validateResource(r, i); err != nil {
			return err
		}

		// Unique name
		if names[r.Name] {
			return fmt.Errorf("config: resources[%d]: duplicate name %q — resource names must be unique", i, r.Name)
		}
		names[r.Name] = true

		// Unique port
		if owner, taken := ports[r.Server.Port]; taken {
			return fmt.Errorf("config: resources[%d] %q: port %d is already used by %q",
				i, r.Name, r.Server.Port, owner)
		}
		ports[r.Server.Port] = r.Name
	}

	return nil
}

func (c *Config) validateGlobal() error {
	if !validLogLevels[strings.ToLower(c.Global.Log.Level)] {
		return fmt.Errorf("config: global.log.level %q is not supported (allowed: debug, info, warn, error)",
			c.Global.Log.Level)
	}
	if !validLogFormats[strings.ToLower(c.Global.Log.Format)] {
		return fmt.Errorf("config: global.log.format %q is not supported (allowed: json, console)",
			c.Global.Log.Format)
	}

	if c.Global.Auth != nil {
		if err := validateAuth("global.auth", c.Global.Auth); err != nil {
			return err
		}
	}
	if c.Global.CORS != nil {
		if err := validateCORS("global.cors", c.Global.CORS); err != nil {
			return err
		}
	}
	return nil
}

func validateResource(r *ResourceConfig, idx int) error {
	loc := func(field string) string {
		return fmt.Sprintf("config: resources[%d] %q: %s", idx, r.Name, field)
	}

	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("config: resources[%d]: name is required", idx)
	}

	if !validResourceTypes[strings.ToLower(r.Type)] {
		return fmt.Errorf("%s type %q is not supported (allowed: database, filestore)", loc(""), r.Type)
	}

	if err := validateServer(loc, &r.Server); err != nil {
		return err
	}

	switch strings.ToLower(r.Type) {
	case "database":
		if r.Database == nil {
			return fmt.Errorf("%s type is 'database' but no 'database:' block is declared", loc(""))
		}
		if r.Filestore != nil {
			return fmt.Errorf("%s type is 'database' but a 'filestore:' block is also present — remove it", loc(""))
		}
		if err := validateDatabase(loc, r.Database); err != nil {
			return err
		}

	case "filestore":
		if r.Filestore == nil {
			return fmt.Errorf("%s type is 'filestore' but no 'filestore:' block is declared", loc(""))
		}
		if r.Database != nil {
			return fmt.Errorf("%s type is 'filestore' but a 'database:' block is also present — remove it", loc(""))
		}
		if err := validateFilestore(loc, r.Filestore); err != nil {
			return err
		}
	}

	// Per-resource optional overrides
	if r.Auth != nil {
		if err := validateAuth(loc("auth"), r.Auth); err != nil {
			return err
		}
	}
	if r.CORS != nil {
		if err := validateCORS(loc("cors"), r.CORS); err != nil {
			return err
		}
	}

	return nil
}

func validateServer(loc func(string) string, s *ServerConfig) error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("%s port %d is out of range (1–65535)", loc("server.port"), s.Port)
	}
	if len(s.Protocols) == 0 {
		return fmt.Errorf("%s must have at least one protocol", loc("server.protocols"))
	}
	for _, p := range s.Protocols {
		if !validProtocols[strings.ToLower(p)] {
			return fmt.Errorf("%s unknown protocol %q (allowed: rest, graphql, grpc, websocket)",
				loc("server.protocols"), p)
		}
	}
	return nil
}

func validateDatabase(loc func(string) string, d *DatabaseConfig) error {
	if !validDBDrivers[strings.ToLower(d.Driver)] {
		return fmt.Errorf("%s driver %q is not supported (allowed: postgres, mysql)", loc("database.driver"), d.Driver)
	}
	if strings.TrimSpace(d.DSN) == "" {
		return fmt.Errorf("%s is required", loc("database.dsn"))
	}
	if d.Pool.MaxConns < 1 {
		return fmt.Errorf("%s must be >= 1", loc("database.pool.max_conns"))
	}
	if d.Pool.MinConns < 0 {
		return fmt.Errorf("%s must be >= 0", loc("database.pool.min_conns"))
	}
	if d.Pool.MinConns > d.Pool.MaxConns {
		return fmt.Errorf("%s (%d) must not exceed max_conns (%d)",
			loc("database.pool.min_conns"), d.Pool.MinConns, d.Pool.MaxConns)
	}
	return nil
}

func validateFilestore(loc func(string) string, f *FilestoreConfig) error {
	if !validFilestoreProviders[strings.ToLower(f.Provider)] {
		return fmt.Errorf("%s provider %q is not supported (allowed: minio, s3)", loc("filestore.provider"), f.Provider)
	}
	if strings.TrimSpace(f.Endpoint) == "" {
		return fmt.Errorf("%s is required", loc("filestore.endpoint"))
	}
	if strings.TrimSpace(f.AccessKey) == "" {
		return fmt.Errorf("%s is required", loc("filestore.access_key"))
	}
	if strings.TrimSpace(f.SecretKey) == "" {
		return fmt.Errorf("%s is required", loc("filestore.secret_key"))
	}
	return nil
}

func validateAuth(location string, a *AuthConfig) error {
	if !a.Enabled {
		return nil
	}
	if !validAuthTypes[strings.ToLower(a.Type)] {
		return fmt.Errorf("config: %s.type %q is not supported (allowed: jwt, apikey)", location, a.Type)
	}
	if strings.ToLower(a.Type) == "jwt" && strings.TrimSpace(a.Secret) == "" {
		return fmt.Errorf("config: %s.secret is required when type is 'jwt'", location)
	}
	return nil
}

func validateCORS(location string, c *CORSConfig) error {
	if !c.Enabled {
		return nil
	}
	if len(c.Origins) == 0 {
		return fmt.Errorf("config: %s.origins must not be empty when cors is enabled", location)
	}
	return nil
}
