package config

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

// Rename detection for hand-edited command files. The app records each
// command's name and content fingerprint in .command_names (app-managed,
// like .last_workflow). When a referenced name disappears while a command
// with the same fingerprint appears under an unknown name, the pair is
// proposed as a rename — the same content-based detection git applies to
// file renames, so the hand-written files stay free of synthetic IDs.

const snapshotFile = ".command_names"

// Rename is a detected old-name → new-name pair, with the reference
// counts shown in the repair offer.
type Rename struct {
	Old, New string
	WfSteps  int // workflow steps referencing Old
	TplRefs  int // derived commands whose template is Old
	VarKeys  int // scoped vars.tsv rows of Old
}

// Fingerprint hashes a command's identity content — everything except
// the name — so a pure rename keeps the fingerprint stable.
func Fingerprint(cmd model.Command) string {
	h := sha256.New()
	field := func(tag, value string) {
		io.WriteString(h, tag)
		io.WriteString(h, "\x00")
		io.WriteString(h, value)
		io.WriteString(h, "\x00")
	}
	field("template", cmd.Template)
	field("cmd", cmd.Cmd)
	field("dir", cmd.Dir)
	field("group", cmd.Group)
	field("shell", cmd.Shell)
	for _, k := range sortedKeys(cmd.Slots) {
		field("slot:"+k, cmd.Slots[k])
	}
	for _, k := range sortedKeys(cmd.Values) {
		field("value:"+k, cmd.Values[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// LoadSnapshot reads the name → fingerprint map recorded by the last
// run. A missing or unreadable file is an empty map: detection simply
// stays quiet until a snapshot exists.
func LoadSnapshot(projectDir string) map[string]string {
	snap := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(projectDir, snapshotFile))
	if err != nil {
		return snap
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if name, fp, ok := strings.Cut(line, "\t"); ok && name != "" && fp != "" {
			snap[name] = fp
		}
	}
	return snap
}

// SaveSnapshot writes the snapshot atomically, sorted by name.
func SaveSnapshot(projectDir string, entries map[string]string) error {
	lines := make([]string, 0, len(entries))
	for _, name := range sortedKeys(entries) {
		lines = append(lines, name+"\t"+entries[name])
	}
	path := filepath.Join(projectDir, snapshotFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DanglingCommandRefs returns the sorted unique command names that
// workflow steps, template fields, or scoped vars.tsv rows reference
// but that no longer resolve to a command.
func DanglingCommandRefs(cfg model.Config, workflows []model.Workflow, vars map[string]string) []string {
	seen := make(map[string]bool)
	var out []string
	dangling := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if _, ok := cfg.FindCommand(name); !ok {
			out = append(out, name)
		}
	}
	for _, wf := range workflows {
		for _, step := range wf.Commands {
			dangling(step)
		}
	}
	for _, cmd := range cfg.Commands {
		if cmd.Template != "" {
			dangling(cmd.Template)
		}
	}
	for k := range vars {
		// Scoped keys are "command.slot" with the separator on the last dot.
		if i := strings.LastIndex(k, "."); i > 0 {
			dangling(k[:i])
		}
	}
	sort.Strings(out)
	return out
}

// DetectRenames matches dangling references against the previous
// snapshot: a missing name whose recorded fingerprint reappears on
// exactly one command unknown to the snapshot is proposed as a rename.
// Ambiguous fingerprints (several candidates, or several missing names
// sharing one) propose nothing — a wrong guess would silently retarget
// workflows, while a skipped one only leaves the existing warnings.
func DetectRenames(cfg model.Config, workflows []model.Workflow, vars map[string]string, snap map[string]string) []Rename {
	var missing []string
	for _, name := range DanglingCommandRefs(cfg, workflows, vars) {
		if _, ok := snap[name]; ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	newByFP := make(map[string][]string)
	for _, cmd := range cfg.AllCommands() {
		if _, known := snap[cmd.Name]; !known {
			fp := Fingerprint(cmd)
			newByFP[fp] = append(newByFP[fp], cmd.Name)
		}
	}
	missingByFP := make(map[string]int)
	for _, name := range missing {
		missingByFP[snap[name]]++
	}

	var renames []Rename
	for _, old := range missing {
		fp := snap[old]
		if candidates := newByFP[fp]; len(candidates) == 1 && missingByFP[fp] == 1 {
			r := Rename{Old: old, New: candidates[0]}
			for _, wf := range workflows {
				for _, step := range wf.Commands {
					if step == old {
						r.WfSteps++
					}
				}
			}
			for _, cmd := range cfg.Commands {
				if cmd.Template == old {
					r.TplRefs++
				}
			}
			prefix := old + "."
			for k := range vars {
				if strings.HasPrefix(k, prefix) && !strings.Contains(k[len(prefix):], ".") {
					r.VarKeys++
				}
			}
			renames = append(renames, r)
		}
	}
	return renames
}
