package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// projectHistoryFile is where the project open order is kept: with the
// other app-managed recency data, so the projects directory gains no file
// of its own.
func projectHistoryFile(projectsDir string) string {
	return filepath.Join(projectsDir, ".baton-recent", "projects.json")
}

// LoadProjectHistory returns project names, most recently opened first.
// A missing or unreadable history is empty.
func LoadProjectHistory(projectsDir string) []string {
	data, err := os.ReadFile(projectHistoryFile(projectsDir))
	if err != nil {
		return nil
	}
	var names []string
	if json.Unmarshal(data, &names) != nil {
		return nil
	}
	return names
}

// RecordProjectOpen moves name to the front of the history and drops
// entries whose project directory no longer exists.
func RecordProjectOpen(projectsDir, name string) error {
	names := []string{name}
	for _, old := range LoadProjectHistory(projectsDir) {
		if old == name || filepath.Base(old) != old {
			continue
		}
		if info, err := os.Stat(filepath.Join(projectsDir, old)); err == nil && info.IsDir() {
			names = append(names, old)
		}
	}
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	path := projectHistoryFile(projectsDir)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".projects-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// SortProjectsByRecent orders names by the history, most recent first;
// projects never opened follow in name order.
func SortProjectsByRecent(projectsDir string, names []string) {
	ranks := map[string]int{}
	for i, name := range LoadProjectHistory(projectsDir) {
		if _, ok := ranks[name]; !ok {
			ranks[name] = i
		}
	}
	sort.SliceStable(names, func(i, j int) bool {
		a, oka := ranks[names[i]]
		b, okb := ranks[names[j]]
		if oka != okb {
			return oka
		}
		if oka && a != b {
			return a < b
		}
		return names[i] < names[j]
	})
}
