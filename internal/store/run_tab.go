package store

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadRunTab returns only supported tabs; missing or stale preferences use the default.
func LoadRunTab(projectDir string) string {
	data, _ := os.ReadFile(filepath.Join(projectDir, ".last_run_tab"))
	tab := strings.TrimSpace(string(data))
	if tab == "workflows" || tab == "commands" {
		return tab
	}
	return ""
}

func SaveRunTab(projectDir, tab string) error {
	if tab != "workflows" && tab != "commands" {
		return nil
	}
	return writeFileAtomic(filepath.Join(projectDir, ".last_run_tab"), []byte(tab))
}
