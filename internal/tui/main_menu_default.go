package tui

// Choose a useful starting action on project load; returning from an action
// continues to use mainMenuCursor so the user's place is preserved.
func (m Model) defaultMainMenuCursor() int {
	action := "Edit project"
	if len(m.config.AllCommands()) > 0 {
		action = "Run commands"
		if len(m.workflows) > 0 {
			action = "Run workflow"
		}
	}
	for i, item := range mainMenuItems() {
		if item == action {
			return i
		}
	}
	return 0
}
