package slot

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var reservedPath = regexp.MustCompile(`\{@[^{}]*\}`)
var extension = regexp.MustCompile(`^\.[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$`)

// RefName keeps ordinary slot keys unchanged. Reserved keys are encoded without
// dots so vars.tsv's command.slot addresses remain unambiguous, even for .tar.gz.
func RefName(ref string) string {
	if strings.HasPrefix(ref, "{@") {
		return "@" + hex.EncodeToString([]byte(ref[1:len(ref)-1]))
	}
	return strings.TrimSuffix(ref[1:len(ref)-1], "...")
}

func PathDefinition(ref string) Def {
	body := ref[2 : len(ref)-1]
	kind, filter, hasFilter := strings.Cut(body, ":")
	d := Def{Name: RefName(ref), Raw: ref, PathKind: strings.TrimSuffix(kind, "..."), Variadic: strings.HasSuffix(kind, "...")}
	if hasFilter {
		d.Extensions = strings.Split(filter, ",")
	}
	return d
}

// PathErrors makes typos in the reserved namespace visible to baton check.
func PathErrors(text string) []string {
	var errors []string
	for _, ref := range reservedPath.FindAllString(text, -1) {
		d := PathDefinition(ref)
		valid := d.PathKind == "file" || d.PathKind == "dir"
		if d.PathKind == "dir" && len(d.Extensions) > 0 {
			valid = false
		}
		for _, ext := range d.Extensions {
			if !extension.MatchString(ext) {
				valid = false
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("invalid path slot %s (use {@file}, {@file...:.go,.mod}, or {@dir})", ref))
		}
	}
	return errors
}
