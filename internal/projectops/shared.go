package projectops

import (
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

func SaveConfig(dir string, cfg mdl.Config, varsIn map[string]string) (map[string]string, error) {
	vars := make(map[string]string, len(varsIn))
	for k, v := range varsIn {
		vars[k] = v
	}
	// Mirror every template-derived command's fixed values into vars.tsv
	// ("command.slot" names), drop entries for commands that no longer
	// exist, and persist the local layer without the values — vars.tsv is
	// their single editable home.
	for _, c := range cfg.Commands {
		if c.Template != "" {
			slot.SetCommandValues(vars, c.Name, c.Values)
		}
	}
	slot.PruneCommandValues(vars, func(name string) bool {
		_, ok := cfg.FindCommand(name)
		return ok
	})
	// Template-derived entries persist only their identity: cmd/workdir/
	// slots/values are all recomputed from template + vars.tsv on load,
	// and a baked copy in the file just looks editable without being so.
	stripped := cfg
	stripped.Commands = make([]mdl.Command, len(cfg.Commands))
	copy(stripped.Commands, cfg.Commands)
	for i := range stripped.Commands {
		if stripped.Commands[i].Template != "" {
			stripped.Commands[i].Values = nil
			stripped.Commands[i].Cmd = ""
			stripped.Commands[i].Dir = ""
			stripped.Commands[i].Slots = nil
		}
	}
	if err := store.SaveConfigWithVars(dir, stripped, vars); err != nil {
		return nil, err
	}
	return vars, nil
}

func RetargetCommandRefs(cfg *mdl.Config, workflows []mdl.Workflow, vars map[string]string, oldName, newName string) (wfSteps, tplRefs, varKeys int) {
	for wi := range workflows {
		for si, step := range workflows[wi].Commands {
			if step == oldName {
				workflows[wi].Commands[si] = newName
				wfSteps++
			}
		}
	}
	for i := range cfg.Commands {
		if cfg.Commands[i].Template == oldName {
			cfg.Commands[i].Template = newName
			tplRefs++
		}
	}
	// Scoped keys are "command.slot" with the separator on the last dot;
	// a key re-keys only when the suffix after oldName is a single segment.
	prefix := oldName + "."
	var moved []string
	for k := range vars {
		if strings.HasPrefix(k, prefix) && !strings.Contains(k[len(prefix):], ".") {
			moved = append(moved, k)
		}
	}
	for _, k := range moved {
		vars[newName+"."+k[len(prefix):]] = vars[k]
		delete(vars, k)
		varKeys++
	}
	return wfSteps, tplRefs, varKeys
}
