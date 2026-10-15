package projectfs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Files map[string][]byte

var fixed = []string{"commands.tsv", "commands.json", "commands.local.json", "workflows.json", "vars.tsv", ".last_workflow", ".command_names"}

func allowed(name string) bool {
	for _, f := range fixed {
		if name == f {
			return true
		}
	}
	return strings.HasPrefix(name, "lists/") && strings.Count(name, "/") == 1 && strings.HasSuffix(name, ".tsv") && !strings.ContainsAny(name, "\\:")
}
func regular(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}
	return nil
}
func Read(dir string) (Files, error) {
	out := Files{}
	for _, name := range fixed {
		path := filepath.Join(dir, name)
		if err := regular(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}
	listDir := filepath.Join(dir, "lists")
	if st, err := os.Lstat(listDir); err == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("lists must be a real directory")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	entries, err := os.ReadDir(listDir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".tsv") {
			continue
		}
		name := "lists/" + e.Name()
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := regular(path); err != nil {
			return nil, err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}
	return out, nil
}

// Revision hashes the definition files. .last_workflow and .command_names
// are bookkeeping the TUI rewrites on its own, so they are left out: a
// running TUI must not make an agent's edit conflict, or the other way
// round.
func Revision(files Files) string {
	keys := make([]string, 0, len(files))
	for k := range files {
		if k == ".last_workflow" || k == ".command_names" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%d:%s:%d:", len(k), k, len(files[k]))
		h.Write(files[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}
func Write(dir, name string, data []byte) error {
	if !allowed(name) {
		return fmt.Errorf("invalid project file %q", name)
	}
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if st, err := os.Lstat(path); err == nil && (!st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("unsafe destination %s", name)
	}
	return atomic(path, data)
}
func atomic(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".baton-write-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Lock takes the exclusive lock of dir for one write. The lock is a file
// holding the owner's pid; a lock whose owner no longer runs is replaced,
// so a crash never needs a cleanup by hand.
func Lock(dir string) (func(), error) {
	path := filepath.Join(dir, ".baton-lock")
	for attempt := 0; ; attempt++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			fmt.Fprintf(f, "pid=%d\n", os.Getpid())
			f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("lock unavailable (%s): %w", path, err)
		}
		pid, ok := lockOwner(path)
		if !ok {
			return nil, fmt.Errorf("project busy: %s exists; remove it only if no baton is running", path)
		}
		// Renaming first makes sure only one of several waiters removes
		// the stale file; the others find it gone and retry.
		if attempt == 0 && !alive(pid) && os.Rename(path, path+".stale") == nil {
			_ = os.Remove(path + ".stale")
			continue
		}
		return nil, fmt.Errorf("project busy: baton process %d holds %s", pid, path)
	}
}

// lockOwner reads the pid recorded in a lock file.
func lockOwner(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var pid int
	if _, err := fmt.Sscanf(string(b), "pid=%d", &pid); err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

const Journal = ".baton-transaction.json"

type journal struct {
	Before Files
	After  Files
}

func Pending(dir string) bool { _, err := os.Stat(filepath.Join(dir, Journal)); return err == nil }
func Changed(before, after Files) []string {
	names := map[string]bool{}
	for k := range before {
		names[k] = true
	}
	for k := range after {
		names[k] = true
	}
	var changed []string
	for k := range names {
		a, ao := before[k]
		b, bo := after[k]
		if ao != bo || string(a) != string(b) {
			changed = append(changed, k)
		}
	}
	sort.Strings(changed)
	return changed
}
func restore(dir string, from, to Files) error {
	for _, name := range Changed(from, to) {
		if !allowed(name) {
			return fmt.Errorf("invalid journal path %q", name)
		}
		if b, ok := to[name]; ok {
			if err := Write(dir, name, b); err != nil {
				return err
			}
		} else {
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
func Commit(dir string, before, after Files) error {
	current, err := Read(dir)
	if err != nil {
		return err
	}
	if Revision(current) != Revision(before) {
		return fmt.Errorf("conflict: project changed while preparing the transaction")
	}
	if Pending(dir) {
		return fmt.Errorf("pending transaction: run project.recover")
	}
	if len(Changed(before, after)) == 0 {
		return nil
	}
	data, err := json.Marshal(journal{before, after})
	if err != nil {
		return err
	}
	if err = atomic(filepath.Join(dir, Journal), data); err != nil {
		return err
	}
	if err = restore(dir, before, after); err != nil {
		now, readErr := Read(dir)
		if readErr != nil {
			return fmt.Errorf("commit failed: %v; recovery needed: %v", err, readErr)
		}
		if rollback := restore(dir, now, before); rollback != nil {
			return fmt.Errorf("commit failed: %v; recovery needed: %v", err, rollback)
		}
		_ = os.Remove(filepath.Join(dir, Journal))
		return err
	}
	return os.Remove(filepath.Join(dir, Journal))
}
func Recover(dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, Journal))
	if err != nil {
		return err
	}
	var j journal
	if err = json.Unmarshal(b, &j); err != nil {
		return err
	}
	for name := range j.Before {
		if !allowed(name) {
			return fmt.Errorf("invalid recovery path")
		}
	}
	for name := range j.After {
		if !allowed(name) {
			return fmt.Errorf("invalid recovery path")
		}
	}
	current, err := Read(dir)
	if err != nil {
		return err
	}
	if Revision(current) != Revision(j.After) {
		for _, name := range Changed(j.Before, j.After) {
			c, co := current[name]
			a, ao := j.Before[name]
			b, bo := j.After[name]
			if !(co == ao && string(c) == string(a)) && !(co == bo && string(c) == string(b)) {
				return fmt.Errorf("recovery conflict in %s; retain journal for manual recovery", name)
			}
		}
		// Restore only transaction files, preserving unrelated later files.
		desired := Files{}
		for k, v := range current {
			desired[k] = v
		}
		for _, name := range Changed(j.Before, j.After) {
			if v, ok := j.Before[name]; ok {
				desired[name] = v
			} else {
				delete(desired, name)
			}
		}
		if err = restore(dir, current, desired); err != nil {
			return err
		}
	}
	return os.Remove(filepath.Join(dir, Journal))
}
