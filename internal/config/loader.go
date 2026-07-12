package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
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

// firstExisting returns the first of names that exists in dir.
func firstExisting(dir string, names ...string) (string, bool) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
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

// LoadConfig loads the hand-written layer (commands.json and/or
// commands.tsv, never written back) and commands.local.json (app-managed
// user commands). Legacy names are still readable: templates.json,
// template.json, templates.tsv, config.tsv, config.json, and the old
// "saved_commands" section (converted to template-derived commands).
func LoadConfig(projectDir string) (model.Config, error) {
	cfg := model.Config{}

	if path, ok := firstExisting(projectDir, "commands.json", "templates.json", "template.json"); ok {
		base, err := loadJSON(path)
		if err != nil {
			return cfg, err
		}
		cfg.Base = base.Commands
	}

	// Loaded unconditionally so that creating commands via the TUI
	// never hides TSV-defined commands.
	if path, ok := firstExisting(projectDir, "commands.tsv", "templates.tsv", "config.tsv"); ok {
		cfgTSV, err := loadTSV(path)
		if err != nil {
			return cfg, err
		}
		cfg.Base = append(cfg.Base, cfgTSV.Commands...)
	}

	if path, ok := firstExisting(projectDir, "commands.local.json", "config.json"); ok {
		cfgJSON, err := loadJSON(path)
		if err != nil {
			return cfg, err
		}
		cfg.Commands = cfgJSON.Commands
	}

	// Re-bake template-derived commands so template edits propagate.
	// Errors are non-fatal: the last baked cmd keeps the entry runnable,
	// and the TUI surfaces missing templates as a warning.
	for i := range cfg.Commands {
		if baked, err := slot.MaterializeCommand(cfg.Commands[i], cfg); err == nil {
			cfg.Commands[i] = baked
		}
	}

	return cfg, nil
}

// legacySavedCommand is the pre-unification "saved_commands" entry shape.
type legacySavedCommand struct {
	Name        string            `json:"name"`
	TemplateRef string            `json:"template_ref"`
	Values      map[string]string `json:"values"`
}

func loadJSON(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, err
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.Config{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	// Convert the legacy "saved_commands" section into template-derived
	// commands. They are migrated to the unified format on next save.
	var legacy struct {
		SavedCommands []legacySavedCommand `json:"saved_commands"`
	}
	if err := json.Unmarshal(data, &legacy); err == nil {
		for _, sc := range legacy.SavedCommands {
			cfg.Commands = append(cfg.Commands, model.Command{
				Name:     sc.Name,
				Template: sc.TemplateRef,
				Values:   sc.Values,
			})
		}
	}
	return cfg, nil
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
			cmd.Slots = parseVarsStr(v)
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
