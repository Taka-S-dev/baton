package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// The UI owns this per-project preference. Empty content means Default;
// absence or unrecognized content means no previous selection is available.
func (m Model) applyShellPreference(cf *commandFormState) {
	if m.projectDir == "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(m.projectDir, ".last_shell"))
	if err != nil {
		return
	}
	value := strings.TrimSpace(string(data))
	if value != "" && value != "ps" {
		return
	}
	cf.fields[4] = value
	cf.shellRemembered = true
}

func (m *Model) rememberShell(value string) {
	if m.projectDir == "" {
		return
	}
	path := filepath.Join(m.projectDir, ".last_shell")
	temp, err := os.CreateTemp(m.projectDir, ".last_shell-*")
	if err == nil {
		name := temp.Name()
		defer os.Remove(name)
		_, err = temp.WriteString(value + "\n")
		closeErr := temp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, path)
		}
	}
	if err != nil {
		m.errMsg = "command saved, but shell preference could not be saved: " + err.Error()
	}
}
