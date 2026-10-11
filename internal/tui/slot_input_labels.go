package tui

import (
	"fmt"
	"strings"
)

func (s *slotPickState) completedPathKind() string {
	if s.completedPath == "" || s.completedPath != s.search {
		return ""
	}
	if strings.HasSuffix(s.search, "/") || strings.HasSuffix(s.search, "\\") {
		return "folder"
	}
	return "file"
}

// completedPathTitle is the kind of the completed path as a label.
func (s *slotPickState) completedPathTitle() string {
	switch s.completedPathKind() {
	case "file":
		return "File"
	case "folder":
		return "Folder"
	}
	return ""
}

func (s *slotPickState) inputHint() string {
	if s.pathDef.PathKind != "" {
		return "/ type to filter · Tab: choices"
	}
	if len(s.entries) == 0 {
		return "/ type a value"
	}
	return "/ filter list or type a value"
}

func (s *slotPickState) inputStatus() string {
	switch s.completedPathKind() {
	case "file":
		return "File path completed"
	case "folder":
		return "Folder path completed"
	}
	if len(s.entries) == 0 {
		return ""
	}
	if len(s.filtered) == 0 {
		return "No list matches"
	}
	return fmt.Sprintf("%d list matches", len(s.filtered))
}

func (s *slotPickState) typedValueLabel() string {
	if s.search == "" {
		return "Type a value above"
	}
	switch s.completedPathKind() {
	case "file":
		return "Use file: " + s.search
	case "folder":
		return "Use folder: " + s.search
	default:
		return "Use value: " + s.search
	}
}
