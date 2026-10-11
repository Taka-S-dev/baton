package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Path focus is a UI preference, separate from command execution history.
func pathFocusFile(projectsDir, base, project, command, slot string) string {
	return filepath.Join(projectsDir, ".baton-recent", "path-focus", recentHash(recentPathKey(base)+"\x00"+recentPathKey(project)+"\x00"+command+"\x00"+slot)+".json")
}

func SavePathFocus(projectsDir, base, project, command, slot, value string) error {
	path := pathFocusFile(projectsDir, base, project, command, slot)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".focus-*")
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

func LoadPathFocus(projectsDir, base, project, command, slot string) string {
	data, err := os.ReadFile(pathFocusFile(projectsDir, base, project, command, slot))
	if err != nil {
		return ""
	}
	var value string
	if json.Unmarshal(data, &value) != nil {
		return ""
	}
	return value
}
