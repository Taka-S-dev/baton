package agent

import "github.com/Taka-S-dev/baton/internal/slot"

// Input is the public wire format. Internal slot field names are not API keys.
type Input struct {
	Name        string   `json:"name"`
	Placeholder string   `json:"placeholder"`
	List        string   `json:"list,omitempty"`
	Variadic    bool     `json:"variadic"`
	PathKind    string   `json:"path_kind,omitempty"`
	Extensions  []string `json:"extensions,omitempty"`
}

func inputSpec(d slot.Def) Input {
	return Input{Name: d.Name, Placeholder: d.Placeholder(), List: d.ListName, Variadic: d.Variadic, PathKind: d.PathKind, Extensions: d.Extensions}
}
func inputSpecs(defs []slot.Def) []Input {
	out := make([]Input, 0, len(defs))
	for _, d := range defs {
		out = append(out, inputSpec(d))
	}
	return out
}
