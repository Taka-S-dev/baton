package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

func LoadWorkflows(projectDir string) ([]model.Workflow, error) {
	var result []model.Workflow
	data, err := os.ReadFile(filepath.Join(projectDir, "workflows.json"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("workflows.json: %w", err)
	}
	return result, nil
}

func SaveWorkflows(projectDir string, workflows []model.Workflow) error {
	data, err := json.MarshalIndent(workflows, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(projectDir, "workflows.json"), data)
}

func LoadLastWorkflow(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, ".last_workflow"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func SaveLastWorkflow(projectDir, name string) {
	_ = os.WriteFile(filepath.Join(projectDir, ".last_workflow"), []byte(name), 0644)
}

// SaveConfig writes app-managed user commands to commands.local.json.
// The hand-written layer (Base) is excluded via the json:"-" tag.
func SaveConfig(projectDir string, cfg model.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(projectDir, "commands.local.json"), data)
}

// writeFileAtomic writes to a temp file then renames it over the target,
// so a crash mid-write never leaves a truncated file.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
