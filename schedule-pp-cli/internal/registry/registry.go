// Package registry persists the user's lane → routePort URL mapping.
// Each entry is a (alias, url) pair the user captured once in their browser.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Route is one user-registered lane.
type Route struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	POL  string `json:"pol,omitempty"`
	POD  string `json:"pod,omitempty"`
	Note string `json:"note,omitempty"`
}

// Store is the on-disk JSON registry.
type Store struct {
	path string
	data struct {
		Routes []Route `json:"routes"`
	}
}

// DefaultPath returns the canonical config path used by the CLI.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "schedule-pp-cli", "routes.json")
}

// Open loads the registry from path; returns an empty store if file is missing.
func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	s := &Store{path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("registry %s: %w", path, err)
	}
	return s, nil
}

// Save writes the registry back to disk, creating directories as needed.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, out, 0o644)
}

// Path returns the on-disk path the registry persists to.
func (s *Store) Path() string { return s.path }

// List returns all routes sorted by name.
func (s *Store) List() []Route {
	out := make([]Route, len(s.data.Routes))
	copy(out, s.data.Routes)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns the route with the given name, or nil if missing.
func (s *Store) Get(name string) *Route {
	for i := range s.data.Routes {
		if strings.EqualFold(s.data.Routes[i].Name, name) {
			return &s.data.Routes[i]
		}
	}
	return nil
}

// Add inserts or replaces a route. URL must contain weiyun001.com/routePort.
func (s *Store) Add(r Route) error {
	if r.Name == "" {
		return fmt.Errorf("route name is required")
	}
	if !strings.Contains(r.URL, "/routePort?") {
		return fmt.Errorf("URL does not look like a Weiyun routePort URL: %s", r.URL)
	}
	for i := range s.data.Routes {
		if strings.EqualFold(s.data.Routes[i].Name, r.Name) {
			s.data.Routes[i] = r
			return nil
		}
	}
	s.data.Routes = append(s.data.Routes, r)
	return nil
}

// Remove deletes a route by name. Returns true if removed.
func (s *Store) Remove(name string) bool {
	for i := range s.data.Routes {
		if strings.EqualFold(s.data.Routes[i].Name, name) {
			s.data.Routes = append(s.data.Routes[:i], s.data.Routes[i+1:]...)
			return true
		}
	}
	return false
}

// Seed registers the default 7 representative lanes if the registry is empty.
// Returns the number of routes added.
func (s *Store) Seed() int {
	if len(s.data.Routes) > 0 {
		return 0
	}
	for _, r := range DefaultRoutes() {
		_ = s.Add(r)
	}
	return len(DefaultRoutes())
}

// DefaultRoutes returns the 7 representative lanes captured during v0.1 build.
// Each URL is stable for the (POL, POD) pair — the encrypted port IDs are
// deterministic across browser sessions.
func DefaultRoutes() []Route {
	return []Route{
		{Name: "NS-JEDDAH", POL: "NANSHA", POD: "JEDDAH", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=A8b%252BKvd%252FrH3knmwPYJlW9A%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=B8CCxPgTyTKHS6UWEi7E0Q%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "NS-SOKHNA", POL: "NANSHA", POD: "SOKHNA", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=p980kImhxnwCsgPmmr5MAw%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=B8CCxPgTyTKHS6UWEi7E0Q%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "NS-KARACHI", POL: "NANSHA", POD: "KARACHI", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=3ez67yisC8fnVI5KVlpklg%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=ZY9R0ulrUtfklWEEetRGPg%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "NS-JEBELALI", POL: "NANSHA", POD: "JEBEL ALI", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=SbjTwb99PCeDQHDeZwEvvA%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=nIJ3Uyv%252Bm7qB2PS5QvkN1A%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "NS-NHAVASHEVA", POL: "NANSHA", POD: "NHAVA SHEVA", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=jD0OvY0B6WYFRxQO7kBFtg%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=ZY9R0ulrUtfklWEEetRGPg%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "NS-DJIBOUTI", POL: "NANSHA", POD: "DJIBOUTI", URL: "https://www.weiyun001.com/routePort?st=huvKGL%252B7Fm4wTC3ZfIhHgw%253D%253D&de=yX7lraUKgIz0E38qoj%252BhZw%253D%253D&rg=Nfupi97Q7g5%252BSThhckCGtg%253D%253D&rt=B8CCxPgTyTKHS6UWEi7E0Q%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
		{Name: "SZ-DJIBOUTI", POL: "SHENZHEN", POD: "DJIBOUTI", URL: "https://www.weiyun001.com/routePort?st=jRrNsSuEnc7RSFFSnVIuRQ%253D%253D&de=yX7lraUKgIz0E38qoj%252BhZw%253D%253D&rg=jRrNsSuEnc7RSFFSnVIuRQ%253D%253D&rt=B8CCxPgTyTKHS6UWEi7E0Q%253D%253D&ap=E414n7wrClEOxW9UQ%252FVHHw%253D%253D&from="},
	}
}
