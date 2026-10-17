package util

import (
	"bytes"
	"regexp"
	"testing"
)

func TestLoggerFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf, LevelDebug)
	Debugf(l, "hello %s", "world")
	pattern := regexp.MustCompile(`^\d{8}-\d{2}:\d{2}:\d{2}\.\d{3} \[DEBUG\] hello world\n$`)
	if !pattern.Match(buf.Bytes()) {
		t.Fatalf("unexpected log line: %q", buf.String())
	}
}
