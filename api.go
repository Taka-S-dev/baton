package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Taka-S-dev/baton/internal/agent"
	"github.com/Taka-S-dev/baton/internal/config"
)

func runAPI(args []string, stdin io.Reader, stdout io.Writer) int {
	write := func(resp agent.Response) int {
		_ = json.NewEncoder(stdout).Encode(resp)
		if !resp.OK {
			return 1
		}
		return 0
	}
	fail := func(err error) int {
		return write(agent.Response{Version: 1, Code: "invalid_request", Error: err.Error()})
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "schema") {
		return write(agent.Response{Version: 1, OK: true, Data: agent.Schema()})
	}
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "--file" {
			return fail(fmt.Errorf("usage: baton api [schema | --file request.json]; otherwise reads JSON from stdin"))
		}
		f, err := os.Open(args[1])
		if err != nil {
			return fail(err)
		}
		defer f.Close()
		stdin = f
	}
	data, err := io.ReadAll(io.LimitReader(stdin, (8<<20)+1))
	if err != nil {
		return fail(err)
	}
	if len(data) > 8<<20 {
		return fail(fmt.Errorf("request exceeds 8 MiB"))
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var r agent.Request
	if err = dec.Decode(&r); err != nil {
		return fail(err)
	}
	var extra any
	if err = dec.Decode(&extra); err != io.EOF {
		return fail(fmt.Errorf("expected one JSON request"))
	}
	if r.Action == "schema" {
		return write(agent.Response{Version: 1, OK: true, Data: agent.Schema()})
	}
	root, err := config.FindProjectsDir()
	if err != nil {
		return fail(err)
	}
	return write((agent.Service{ProjectsDir: root}).Handle(r))
}
