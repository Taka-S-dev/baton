package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/projectfs"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// commit runs one write of the open project under the project's lock,
// the same lock the API takes, and refuses it when the definition files
// changed since the project was opened — by another baton, an agent or
// an editor — so a stale screen never overwrites those edits. The TUI's
// own writes move the revision along.
func (m *Model) commit(write func() error) error {
	if m.projectDir == "" {
		return fmt.Errorf("no project is open")
	}
	release, err := projectfs.Lock(m.projectDir)
	if err != nil {
		return err
	}
	defer release()
	if m.revision != "" && revisionOf(m.projectDir) != m.revision {
		return fmt.Errorf("project files changed on disk since this project was opened; reopen it to edit")
	}
	err = write()
	m.revision = revisionOf(m.projectDir)
	return err
}

// revisionOf hashes the definition files of a project; "" when they
// cannot be read, which turns the stale check off for that project.
func revisionOf(dir string) string {
	files, err := projectfs.Read(dir)
	if err != nil {
		return ""
	}
	return projectfs.Revision(files)
}

func (m *Model) saveWorkflows() error {
	return m.commit(func() error { return store.SaveWorkflows(m.projectDir, m.workflows) })
}

func (m *Model) saveVars() error {
	return m.commit(func() error { return slot.SaveVars(m.projectDir, m.vars) })
}

func (m *Model) saveList(name string, entries []mdl.ListEntry) error {
	return m.commit(func() error { return slot.SaveList(filepath.Join(m.projectDir, "lists"), name, entries) })
}

func (m *Model) removeList(name string) error {
	return m.commit(func() error { return os.Remove(filepath.Join(m.projectDir, "lists", name+".tsv")) })
}

func (m *Model) appendCommandTSV(c mdl.Command) (file string, err error) {
	err = m.commit(func() (e error) { file, e = config.AppendCommandTSV(m.projectDir, c); return e })
	return file, err
}

func (m *Model) updateCommandTSV(oldName string, c mdl.Command) (file string, err error) {
	err = m.commit(func() (e error) { file, e = config.UpdateCommandTSV(m.projectDir, oldName, c); return e })
	return file, err
}

func (m *Model) deleteCommandsTSV(names []string) error {
	return m.commit(func() error { _, err := config.DeleteCommandsTSV(m.projectDir, names); return err })
}
