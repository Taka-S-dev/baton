package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// SaveConfigWithVars prepares both files before replacing either, and restores
// earlier replacements if a later one fails. This handles I/O errors, not
// process termination or power loss between replacements.
func SaveConfigWithVars(projectDir string, cfg model.Config, vars map[string]string) error {
	return saveConfigWithVars(projectDir, cfg, vars, os.Rename)
}

func saveConfigWithVars(projectDir string, cfg model.Config, vars map[string]string, rename func(string, string) error) error {
	stage, err := os.MkdirTemp(projectDir, ".baton-save-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(stage)
		}
	}()

	names := []string{"vars.tsv", "commands.local.json"}
	type original struct {
		exists bool
		mode   os.FileMode
	}
	originals := make(map[string]original)
	for _, name := range names {
		path := filepath.Join(projectDir, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s: expected a regular file", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(stage, name+".backup"), data, info.Mode().Perm()); err != nil {
			return err
		}
		originals[name] = original{true, info.Mode().Perm()}
	}
	// Preserve SaveVars' rule: an empty map creates no new file, but empties
	// an existing one. The staged empty file makes that distinction explicit.
	if originals["vars.tsv"].exists {
		if err := os.WriteFile(filepath.Join(stage, "vars.tsv"), nil, 0600); err != nil {
			return err
		}
	}
	if err := slot.SaveVars(stage, vars); err != nil {
		return err
	}
	if err := SaveConfig(stage, cfg); err != nil {
		return err
	}
	var pending []string
	for _, name := range names {
		path := filepath.Join(stage, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if old := originals[name]; old.exists {
			if err := os.Chmod(path, old.mode); err != nil {
				return err
			}
		}
		pending = append(pending, name)
	}
	var committed []string
	for _, name := range pending {
		if err := rename(filepath.Join(stage, name), filepath.Join(projectDir, name)); err != nil {
			saveErr := fmt.Errorf("save %s: %w", name, err)
			for i := len(committed) - 1; i >= 0; i-- {
				previous := committed[i]
				target := filepath.Join(projectDir, previous)
				var rollbackErr error
				if originals[previous].exists {
					rollbackErr = rename(filepath.Join(stage, previous+".backup"), target)
				} else {
					rollbackErr = os.Remove(target)
				}
				if rollbackErr != nil {
					keepBackup = true
					saveErr = errors.Join(saveErr, fmt.Errorf("restore %s: %w; recovery files kept in %s", previous, rollbackErr, stage))
				}
			}
			return saveErr
		}
		committed = append(committed, name)
	}
	return nil
}
