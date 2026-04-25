package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

func LoadWorkflows(projectDir string) []model.Workflow {
	var result []model.Workflow
	if data, err := os.ReadFile(filepath.Join(projectDir, "workflows.json")); err == nil {
		_ = json.Unmarshal(data, &result)
	}
	return result
}

func SaveWorkflows(projectDir string, workflows []model.Workflow) error {
	data, err := json.MarshalIndent(workflows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, "workflows.json"), data, 0644)
}

func LoadAliases(projectDir string) []model.Alias {
	var result []model.Alias
	if data, err := os.ReadFile(filepath.Join(projectDir, "aliases.json")); err == nil {
		_ = json.Unmarshal(data, &result)
	}
	return result
}

func SaveAliases(projectDir string, aliases []model.Alias) error {
	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, "aliases.json"), data, 0644)
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
