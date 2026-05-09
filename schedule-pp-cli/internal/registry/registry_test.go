package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedExamples(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "routes.json"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	added := s.Seed()
	if added == 0 {
		t.Fatal("seed should add example routes on empty registry")
	}
	if len(s.List()) != 7 {
		t.Errorf("seeded list count = %d, want 7", len(s.List()))
	}
	// Idempotent: second seed is a no-op.
	if s.Seed() != 0 {
		t.Error("seed should be idempotent on non-empty registry")
	}
}

func TestExampleRoutesShape(t *testing.T) {
	rs := ExampleRoutes()
	if len(rs) != 7 {
		t.Fatalf("ExampleRoutes count = %d, want 7", len(rs))
	}
	for _, r := range rs {
		if r.Name == "" {
			t.Errorf("example route missing name: %+v", r)
		}
		if r.POL == "" || r.POD == "" {
			t.Errorf("example route %q missing POL/POD", r.Name)
		}
		if !strings.Contains(r.URL, "/routePort?") {
			t.Errorf("example route %q has non-routePort URL: %s", r.Name, r.URL)
		}
	}
}

func TestAddGetRemove(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "routes.json"))
	r := Route{Name: "NS-XX", POL: "NANSHA", POD: "X", URL: "https://www.weiyun001.com/routePort?st=A&de=B"}
	if err := s.Add(r); err != nil {
		t.Fatalf("add: %v", err)
	}
	got := s.Get("ns-xx") // case-insensitive
	if got == nil || got.POL != "NANSHA" {
		t.Errorf("get failed: %+v", got)
	}
	if !s.Remove("NS-XX") {
		t.Error("remove returned false")
	}
	if s.Get("NS-XX") != nil {
		t.Error("get should return nil after remove")
	}
}

func TestAddRejectsBadURL(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "routes.json"))
	err := s.Add(Route{Name: "X", URL: "https://example.com/foo"})
	if err == nil {
		t.Error("expected error for non-routePort URL")
	}
	err = s.Add(Route{Name: "", URL: "https://www.weiyun001.com/routePort?x=1"})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.json")
	s1, _ := Open(path)
	s1.Seed()
	if err := s1.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(s2.List()) != 7 {
		t.Errorf("reload count = %d, want 7", len(s2.List()))
	}
	if s2.Get("NS-JEDDAH") == nil {
		t.Error("NS-JEDDAH should round-trip")
	}
}

func TestUpdateReplaces(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "routes.json"))
	_ = s.Add(Route{Name: "X", URL: "https://www.weiyun001.com/routePort?v=1", Note: "first"})
	_ = s.Add(Route{Name: "X", URL: "https://www.weiyun001.com/routePort?v=2", Note: "second"})
	if len(s.List()) != 1 {
		t.Errorf("re-add should replace, got %d entries", len(s.List()))
	}
	if s.Get("X").Note != "second" {
		t.Error("re-add should overwrite previous note")
	}
}
