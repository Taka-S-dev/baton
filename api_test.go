package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIJSONTransport(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BATON_PROJECTS_DIR", root)
	for _, input := range []string{
		"\ufeff" + `{"action":"project.create","project":"demo"}`,
		`{"action":"project.get","project":"demo"}`,
		`{"action":"schema"}`,
	} {
		var out bytes.Buffer
		if exit := runAPI(nil, strings.NewReader(input), &out); exit != 0 {
			t.Fatal(out.String())
		}
		var response map[string]any
		if err := json.Unmarshal(out.Bytes(), &response); err != nil || response["ok"] != true {
			t.Fatal(out.String(), err)
		}
	}
	for _, input := range []string{`{"action":"project.get","project":"demo","typo":true}`, `{"action":"schema"} {}`, `{"action":"apply","project":"demo","revision":"old","operations":[{"op":"variable.create","name":"x","value":"v","typo":1}]}`} {
		var out bytes.Buffer
		if runAPI(nil, strings.NewReader(input), &out) == 0 {
			t.Fatal("accepted invalid JSON request")
		}
		var response map[string]any
		if err := json.Unmarshal(out.Bytes(), &response); err != nil || response["ok"] != false {
			t.Fatal(out.String(), err)
		}
	}
	file := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(file, []byte(`{"action":"project.get","project":"demo"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if runAPI([]string{"--file", file}, strings.NewReader(""), &out) != 0 {
		t.Fatal(out.String())
	}
}
