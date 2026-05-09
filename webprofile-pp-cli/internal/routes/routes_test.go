package routes

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.toml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load missing should not error: %v", err)
	}
	if !cfg.IsEmpty() {
		t.Errorf("missing file should yield empty config, got %+v", cfg.Routes.Covered)
	}
	if cfg.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %d, want %d", cfg.SchemaVersion, SchemaVersion)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdir", "routes.toml") // exercises MkdirAll
	cfg := New()
	if _, err := cfg.Add("egy"); err != nil {
		t.Fatalf("add EGY: %v", err)
	}
	if _, err := cfg.Add("SAU"); err != nil {
		t.Fatalf("add SAU: %v", err)
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !loaded.IsCovered("EGY") || !loaded.IsCovered("SAU") {
		t.Errorf("roundtrip lost entries: %+v", loaded.Routes.Covered)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version after reload = %d, want %d", loaded.SchemaVersion, SchemaVersion)
	}
}

func TestSaveAtomicAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.toml")
	cfg := New()
	_, _ = cfg.Add("EGY")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode = %o, want 0600", mode)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error(".tmp file should be removed after successful save")
	}
}

func TestIsCoveredCaseInsensitive(t *testing.T) {
	cfg := New()
	_, _ = cfg.Add("egy")
	cases := map[string]bool{
		"EGY":   true,
		"egy":   true,
		" EGY ": true,
		"Egy":   true,
		"USA":   false,
		"":      false,
		"EG":    false,
	}
	for q, want := range cases {
		if got := cfg.IsCovered(q); got != want {
			t.Errorf("IsCovered(%q) = %v, want %v", q, got, want)
		}
	}
}

func TestAddIdempotent(t *testing.T) {
	cfg := New()
	added, err := cfg.Add("EGY")
	if err != nil || !added {
		t.Fatalf("first add: added=%v err=%v", added, err)
	}
	added, err = cfg.Add("egy")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if added {
		t.Error("re-adding (case differ) should report not-added")
	}
	if len(cfg.Routes.Covered) != 1 {
		t.Errorf("duplicate stored: %+v", cfg.Routes.Covered)
	}
}

func TestAddRejectsBadInput(t *testing.T) {
	cfg := New()
	bad := []string{"", "EG", "EGYPT", "Egyptian", "12A", " ", "E_Y"}
	for _, in := range bad {
		if _, err := cfg.Add(in); err == nil {
			t.Errorf("Add(%q) should error", in)
		}
	}
}

func TestRemove(t *testing.T) {
	cfg := New()
	_, _ = cfg.Add("EGY")
	_, _ = cfg.Add("SAU")
	if !cfg.Remove("egy") {
		t.Error("remove case-insensitive should succeed")
	}
	if cfg.IsCovered("EGY") {
		t.Error("EGY should be gone")
	}
	if cfg.Remove("XXX") {
		t.Error("remove non-existent should return false")
	}
	if len(cfg.Routes.Covered) != 1 {
		t.Errorf("count after remove = %d, want 1", len(cfg.Routes.Covered))
	}
}

func TestInit(t *testing.T) {
	cfg := New()
	_, _ = cfg.Add("OLD")
	if err := cfg.Init([]string{"egy", "SAU", "egy", "ARE"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(cfg.Routes.Covered) != 3 {
		t.Errorf("dedupe failed, got %+v", cfg.Routes.Covered)
	}
	if cfg.IsCovered("OLD") {
		t.Error("init should replace, not merge")
	}
	if !slices.IsSorted(cfg.Routes.Covered) {
		t.Errorf("init result not sorted: %+v", cfg.Routes.Covered)
	}
}

func TestInitRejectsBadInput(t *testing.T) {
	cfg := New()
	if err := cfg.Init([]string{"EGY", "Egyptian"}); err == nil {
		t.Error("init with bad code should error")
	}
}

func TestReset(t *testing.T) {
	cfg := New()
	_, _ = cfg.Add("EGY")
	cfg.Reset()
	if !cfg.IsEmpty() {
		t.Error("reset should clear list")
	}
	if cfg.SchemaVersion != SchemaVersion {
		t.Errorf("reset should preserve schema_version, got %d", cfg.SchemaVersion)
	}
}

func TestNormalizeOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.toml")
	raw := `schema_version = 1
[routes]
covered = ["egy", "  SAU ", "EGY", "are"]
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := []string{"ARE", "EGY", "SAU"}
	if !slices.Equal(cfg.Routes.Covered, want) {
		t.Errorf("normalized = %v, want %v", cfg.Routes.Covered, want)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.toml")
	if err := os.WriteFile(path, []byte("this is not = valid toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("invalid TOML should error")
	}
}

func TestDefaultPathIsAbsolute(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p) {
		t.Errorf("DefaultPath = %q, want absolute", p)
	}
}
