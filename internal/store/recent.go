package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// RecentEntry stores references only, never commands or input values.
type RecentEntry struct {
	Directory string
	Project   string
	Command   string
	UsedAt    time.Time
}

func recentPathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
func recentHash(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
func recentDirectory(projectsDir, cwd string) string {
	return filepath.Join(projectsDir, ".baton-recent", recentHash(recentPathKey(cwd)))
}

// Independent references have separate atomic files to avoid lost updates
// when multiple baton processes run different commands concurrently.
func RecordRecent(projectsDir, cwd, project, command string, now time.Time) error {
	if projectsDir == "" || cwd == "" || project == "" || command == "" {
		return fmt.Errorf("incomplete recent command reference")
	}
	var err error
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return err
	}
	project, err = filepath.Abs(project)
	if err != nil {
		return err
	}
	dir := recentDirectory(projectsDir, cwd)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(RecentEntry{cwd, project, command, now})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".write-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, recentHash(recentPathKey(project)+"\x00"+command)+".json"))
}
func LoadRecent(projectsDir, cwd string) ([]RecentEntry, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	dir := recentDirectory(projectsDir, cwd)
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []RecentEntry
	var firstErr error
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		var entry RecentEntry
		if err == nil {
			err = json.Unmarshal(data, &entry)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("recent history: %w", err)
			}
			continue
		}
		if recentPathKey(entry.Directory) != recentPathKey(cwd) || !filepath.IsAbs(entry.Project) || entry.Command == "" {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].UsedAt.Equal(entries[j].UsedAt) {
			return entries[i].Project+entries[i].Command < entries[j].Project+entries[j].Command
		}
		return entries[i].UsedAt.After(entries[j].UsedAt)
	})
	if len(entries) > 100 {
		entries = entries[:100]
	}
	return entries, firstErr
}
