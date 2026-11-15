package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	var file bytes.Buffer
	buf := NewBuffer(2)
	log := New(&file, buf)

	log.Info("one")
	log.Warn("two", "host", "web1")
	log.Info("three")

	if n := strings.Count(file.String(), "\n"); n != 3 {
		t.Errorf("file has %d lines, want 3:\n%s", n, file.String())
	}
	if !strings.Contains(file.String(), "time=") {
		t.Error("file records have no time")
	}

	lines, total := buf.Lines()
	if total != 3 || len(lines) != 2 {
		t.Fatalf("Lines() = %q, %d", lines, total)
	}
	if lines[0] != "level=WARN msg=two host=web1" || lines[1] != "level=INFO msg=three" {
		t.Errorf("Lines() = %q", lines)
	}
}
