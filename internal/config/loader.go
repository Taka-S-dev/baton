package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		for i := range base.Commands {
			base.Commands[i].Source = "json"
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
		for i := range cfgTSV.Commands {
			cfgTSV.Commands[i].Source = "tsv"
		}
		cfg.Base = append(cfg.Base, cfgTSV.Commands...)
	}

	if path, ok := firstExisting(projectDir, "commands.local.json", "config.json"); ok {
		cfgJSON, err := loadJSON(path)
		if err != nil {
			return cfg, err
		}
		for i := range cfgJSON.Commands {
			cfgJSON.Commands[i].Source = "local"
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

// tsvRow serializes a command as a TSV data row.
func tsvRow(cmd model.Command) string {
	var slots []string
	for k, v := range cmd.Slots {
		slots = append(slots, k+"="+v)
	}
	sort.Strings(slots)
	return strings.Join([]string{cmd.Name, cmd.Group, cmd.Dir, cmd.Cmd, cmd.Shell, strings.Join(slots, ",")}, "\t")
}

// tsvRowName returns the command name of a TSV data line.
func tsvRowName(line string) string {
	name := line
	if i := strings.IndexByte(line, '\t'); i >= 0 {
		name = line[:i]
	}
	return unquoteTSVField(strings.TrimSpace(name))
}

// writeTSVLines writes lines back to path atomically, joined with the
// file's original line ending.
func writeTSVLines(path, eol string, lines []string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, eol)+eol), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readTSVLines reads path fresh (so edits saved by an editor moments
// ago are preserved) and returns its logical lines plus the detected
// line ending. A missing file yields just a header line.
func readTSVLines(path string) (lines []string, eol string) {
	eol = "\n"
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{"name\tgroup\tworkdir\tcmd\tshell\tslots"}, eol
	}
	raw := string(data)
	if strings.Contains(raw, "\r\n") {
		eol = "\r\n"
	}
	lines = strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = []string{"name\tgroup\tworkdir\tcmd\tshell\tslots"}
	}
	return lines, eol
}

// AppendCommandTSV appends a command as a new row to the project's
// hand-written TSV — the file the loader actually reads (commands.tsv,
// or a legacy name if that is what the project uses), created with a
// header when none exists. Existing rows are never modified. Returns
// the file name written to.
func AppendCommandTSV(projectDir string, cmd model.Command) (string, error) {
	path, ok := firstExisting(projectDir, "commands.tsv", "templates.tsv", "config.tsv")
	if !ok {
		path = filepath.Join(projectDir, "commands.tsv")
	}
	lines, eol := readTSVLines(path)
	lines = append(lines, tsvRow(cmd))
	if err := writeTSVLines(path, eol, lines); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}

// UpdateCommandTSV replaces the row named oldName with cmd. Every other
// line is left byte-for-byte untouched. Returns the file name, or an
// error when the row no longer exists (edited by hand since load).
func UpdateCommandTSV(projectDir, oldName string, cmd model.Command) (string, error) {
	path, ok := firstExisting(projectDir, "commands.tsv", "templates.tsv", "config.tsv")
	if !ok {
		return "", fmt.Errorf("no TSV command file in this project")
	}
	lines, eol := readTSVLines(path)
	found := false
	for i := 1; i < len(lines); i++ {
		if tsvRowName(lines[i]) == oldName {
			lines[i] = tsvRow(cmd)
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("command %q not found in %s (changed on disk? re-open the project)", oldName, filepath.Base(path))
	}
	if err := writeTSVLines(path, eol, lines); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}

// DeleteCommandsTSV removes the rows with the given names. Names whose
// row is already gone are ignored (deleted by hand is deleted).
func DeleteCommandsTSV(projectDir string, names []string) (string, error) {
	path, ok := firstExisting(projectDir, "commands.tsv", "templates.tsv", "config.tsv")
	if !ok {
		return "", fmt.Errorf("no TSV command file in this project")
	}
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}
	lines, eol := readTSVLines(path)
	kept := lines[:1]
	for _, line := range lines[1:] {
		if !drop[tsvRowName(line)] {
			kept = append(kept, line)
		}
	}
	if err := writeTSVLines(path, eol, kept); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
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
				return unquoteTSVField(strings.TrimSpace(parts[idx]))
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

// unquoteTSVField strips Excel-style field quoting: Excel wraps cells that
// contain commas in double quotes and doubles embedded quotes when saving
// as TSV ("a,b" / "say ""hi""").
func unquoteTSVField(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
	}
	return s
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
