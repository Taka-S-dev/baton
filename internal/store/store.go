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
	return os.WriteFile(filepath.Join(projectDir, "workflows.json"), data, 0644)
}

func LoadAliases(projectDir string) ([]model.Alias, error) {
	var result []model.Alias
	data, err := os.ReadFile(filepath.Join(projectDir, "aliases.json"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("aliases.json: %w", err)
	}
	return result, nil
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
