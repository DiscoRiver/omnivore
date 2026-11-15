package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/discoriver/omnivore/internal/state"
)

// PrintSummary writes the grouped results in snap to w as plain text, with terminal escape
// sequences removed from the output.
func PrintSummary(w io.Writer, snap state.Snapshot) error {
	count := func(s state.Status) int {
		return len(snap.Names(func(h state.Host) bool { return h.Status == s }))
	}
	slow := len(snap.Names(func(h state.Host) bool { return h.Slow }))

	var b strings.Builder
	fmt.Fprintf(&b, "Command: %s\n", snap.Command)
	fmt.Fprintf(&b, "Hosts: %d completed, %d failed, %d slow, %d groups\n",
		count(state.Completed), count(state.Failed), slow, len(snap.Groups))
	for _, grp := range snap.Groups {
		fmt.Fprintf(&b, "\n== %d host(s): %s\n", len(grp.Hosts), strings.Join(grp.Hosts, ", "))
		// Strip terminal escape sequences; the raw output is in the history files.
		out := sanitize(grp.Output)
		b.WriteString(out)
		if out != "" && !strings.HasSuffix(out, "\n") {
			b.WriteByte('\n')
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
