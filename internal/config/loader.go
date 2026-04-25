package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

// FindProjectsDir locates the projects/ directory.
// Resolution order:
//  1. $BATON_PROJECTS_DIR environment variable
//  2. projects/ adjacent to the executable (portable / Windows install)
//  3. ~/.config/baton/projects/ (XDG, Linux/macOS system installs)
func FindProjectsDir() (string, error) {
	if v := os.Getenv("BATON_PROJECTS_DIR"); v != "" {
		if _, err := os.Stat(v); err != nil {
			return "", fmt.Errorf("BATON_PROJECTS_DIR=%q: %w", v, err)
		}
		return v, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if dir := filepath.Join(filepath.Dir(exe), "projects"); dirExists(dir) {
		return dir, nil
	}

	if home, err := os.UserHomeDir(); err == nil {
		if dir := filepath.Join(home, ".config", "baton", "projects"); dirExists(dir) {
			return dir, nil
		}
	}

	return "", fmt.Errorf("projects/ directory not found (set BATON_PROJECTS_DIR to specify a location)")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ListProjects returns subdirectory names inside projectsDir.
func ListProjects(projectsDir string) []string {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// LoadConfig loads config.json (preferred) or config.tsv from projectDir.
func LoadConfig(projectDir string) (model.Config, error) {
	if _, err := os.Stat(filepath.Join(projectDir, "config.json")); err == nil {
		return loadJSON(filepath.Join(projectDir, "config.json"))
	}
	if _, err := os.Stat(filepath.Join(projectDir, "config.tsv")); err == nil {
		return loadTSV(filepath.Join(projectDir, "config.tsv"))
	}
	return model.Config{}, nil
}

func loadJSON(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, err
	}
	var cfg model.Config
	return cfg, json.Unmarshal(data, &cfg)
}

func loadTSV(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var commands []model.Command
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		get := func(idx int) string {
			if idx < len(parts) {
				return strings.TrimSpace(parts[idx])
			}
			return ""
		}
		cmd := model.Command{
			Name:  get(0),
			Group: get(1),
			Dir:   get(2),
			Cmd:   get(3),
			Shell: get(4),
		}
		if v := get(5); v != "" {
			cmd.Vars = parseVarsStr(v)
		}
		if cmd.Name != "" && cmd.Cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return model.Config{Commands: commands}, nil
}

func parseVarsStr(s string) map[string]string {
	result := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
