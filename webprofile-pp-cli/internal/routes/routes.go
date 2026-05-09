// Package routes manages the user's covered-route configuration: the list
// of ISO3 country codes the user can ship freight to.
//
// Boundary: this package owns per-user business config. It knows nothing
// about UN Comtrade, fit-score math, or any external data source. The
// comtrade package owns trade data; neither package imports the other.
// The cli layer wires them together.
package routes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const SchemaVersion = 1

// Config is the persisted shape of routes.toml.
type Config struct {
	SchemaVersion int    `toml:"schema_version"`
	Routes        Routes `toml:"routes"`
}

// Routes holds the covered-route data. Kept as a nested table so future
// fields (weights, scoring overrides) can land here without breaking the
// top-level schema.
type Routes struct {
	Covered []string `toml:"covered"`
}

// New returns an empty Config tagged with the current schema version.
func New() *Config {
	return &Config{SchemaVersion: SchemaVersion}
}

// DefaultPath returns ~/.config/webprofile-pp-cli/routes.toml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "webprofile-pp-cli", "routes.toml"), nil
}

// Load reads the config from path. A missing file is not an error — it
// returns an empty Config so callers can treat first-run and configured
// users uniformly.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := New()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.normalize()
	return cfg, nil
}

// LoadDefault reads from DefaultPath and returns the resolved path so the
// caller can show it to the user or pass it back to Save.
func LoadDefault() (*Config, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := Load(path)
	return cfg, path, err
}

// Save writes the config atomically (write to .tmp + rename) and creates
// parent directories if needed. File mode is 0600 — config is per-user.
func (c *Config) Save(path string) error {
	c.normalize()
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// IsCovered reports whether iso3 is in the covered list. Case-insensitive
// and whitespace-tolerant.
func (c *Config) IsCovered(iso3 string) bool {
	norm := normalizeISO3(iso3)
	if norm == "" {
		return false
	}
	return slices.Contains(c.Routes.Covered, norm)
}

// IsEmpty reports whether the covered list has no entries.
func (c *Config) IsEmpty() bool {
	return len(c.Routes.Covered) == 0
}

// Add inserts an ISO3 if not already present. Returns (added, error):
// added=false with nil error means the entry was a duplicate (idempotent).
func (c *Config) Add(iso3 string) (bool, error) {
	norm, err := validateISO3(iso3)
	if err != nil {
		return false, err
	}
	if slices.Contains(c.Routes.Covered, norm) {
		return false, nil
	}
	c.Routes.Covered = append(c.Routes.Covered, norm)
	slices.Sort(c.Routes.Covered)
	return true, nil
}

// Remove deletes an ISO3 if present. Returns true when removed.
func (c *Config) Remove(iso3 string) bool {
	norm := normalizeISO3(iso3)
	if norm == "" {
		return false
	}
	idx := slices.Index(c.Routes.Covered, norm)
	if idx < 0 {
		return false
	}
	c.Routes.Covered = slices.Delete(c.Routes.Covered, idx, idx+1)
	return true
}

// Init replaces the covered list with the given codes (validated, deduped,
// sorted). Useful for first-time setup or scripted redeployment.
func (c *Config) Init(iso3s []string) error {
	seen := make(map[string]struct{}, len(iso3s))
	out := make([]string, 0, len(iso3s))
	for _, raw := range iso3s {
		norm, err := validateISO3(raw)
		if err != nil {
			return err
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	slices.Sort(out)
	c.Routes.Covered = out
	return nil
}

// Reset clears the covered list. Schema version is preserved so the file
// stays parseable after reset.
func (c *Config) Reset() {
	c.Routes.Covered = nil
}

// normalize sorts/dedupes/uppercases entries so the on-disk shape is
// canonical regardless of how a user hand-edited the file.
func (c *Config) normalize() {
	if c == nil || c.Routes.Covered == nil {
		return
	}
	seen := make(map[string]struct{}, len(c.Routes.Covered))
	out := c.Routes.Covered[:0]
	for _, code := range c.Routes.Covered {
		norm := normalizeISO3(code)
		if norm == "" {
			continue
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	slices.Sort(out)
	c.Routes.Covered = out
}

func normalizeISO3(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// validateISO3 returns a normalized 3-letter alphabetic code. Membership in
// the actual ISO 3166-1 list is intentionally not enforced here — Comtrade's
// reporters file is the authoritative country source for fit-score lookups,
// and tying routes to that would couple the packages. A shape check is
// enough to catch typos like "Egyptian" or "EG".
func validateISO3(s string) (string, error) {
	norm := normalizeISO3(s)
	if len(norm) != 3 {
		return "", fmt.Errorf("%q is not a 3-letter ISO3 code (e.g. EGY)", s)
	}
	for _, r := range norm {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%q contains non-letter characters; ISO3 codes are 3 letters (e.g. EGY)", s)
		}
	}
	return norm, nil
}
