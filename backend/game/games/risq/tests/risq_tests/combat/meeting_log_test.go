package combat

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"testing"

	"github.com/dgray001/gray_online/util"
)

type meetingLine struct{ unit, target, tick, turn, sunk, attack int }
type hitLine struct {
	attacker, target, tick, stamina int
	damage                          float64
}
type moveLine struct{ unit, tick, cost int }

type tickLog struct {
	meetings []meetingLine
	hits     []hitLine
	moves    []moveLine
}

var (
	meetingRe = regexp.MustCompile(`Meeting at border: unit (\d+) meets (\d+) tick=(\d+) turn=(\d+) sunk=(\d+) attack=(\d+)`)
	hitRe     = regexp.MustCompile(`combat turn=\d+ tick=(\d+): (\d+) \w+ \(player \d, stamina (\d+)\) hits (\d+) \w+ \(player \d\) for ([\d.]+)`)
	moveRe    = regexp.MustCompile(`Moving unit (\d+) .* tick=(\d+) turn=\d+ stamina=(\d+)`)
)

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Routes the engine's debug log into a buffer for the test, so tick-level behavior can be asserted
func captureTicks(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	util.DebugLog.SetOutput(&buf)
	t.Cleanup(func() { util.DebugLog.SetOutput(io.Discard) })
	return &buf
}

func parseTicks(t *testing.T, buf *bytes.Buffer) tickLog {
	t.Helper()
	var out tickLog
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if m := meetingRe.FindSubmatch(line); m != nil {
			out.meetings = append(out.meetings, meetingLine{atoi(t, string(m[1])), atoi(t, string(m[2])), atoi(t, string(m[3])), atoi(t, string(m[4])), atoi(t, string(m[5])), atoi(t, string(m[6]))})
		} else if m := hitRe.FindSubmatch(line); m != nil {
			damage, _ := strconv.ParseFloat(string(m[5]), 64)
			out.hits = append(out.hits, hitLine{atoi(t, string(m[2])), atoi(t, string(m[4])), atoi(t, string(m[1])), atoi(t, string(m[3])), damage})
		} else if m := moveRe.FindSubmatch(line); m != nil {
			out.moves = append(out.moves, moveLine{atoi(t, string(m[1])), atoi(t, string(m[2])), atoi(t, string(m[3]))})
		}
	}
	return out
}

// Hits one unit landed, in tick order
func (l tickLog) hitsBy(unit int) []hitLine {
	var hits []hitLine
	for _, h := range l.hits {
		if h.attacker == unit {
			hits = append(hits, h)
		}
	}
	return hits
}

func (l tickLog) movesOf(unit int) []moveLine {
	var moves []moveLine
	for _, m := range l.moves {
		if m.unit == unit {
			moves = append(moves, m)
		}
	}
	return moves
}

func (l tickLog) meetingOf(unit int) (meetingLine, bool) {
	for _, m := range l.meetings {
		if m.unit == unit {
			return m, true
		}
	}
	return meetingLine{}, false
}
