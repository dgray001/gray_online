package util

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var perfEnabled atomic.Bool
var perfMu sync.Mutex
var perfFile *os.File
var perfStdout atomic.Bool

func PerfEnabled() bool {
	return perfEnabled.Load()
}

func EnablePerfToStdout() {
	perfStdout.Store(true)
	perfEnabled.Store(true)
}

func PerfToStdout() bool {
	return perfStdout.Load()
}

// Enabling opens a new logs/perf_<time>.jsonl; disabling closes it
func SetPerfEnabled(enabled bool) error {
	perfMu.Lock()
	defer perfMu.Unlock()
	if !enabled {
		perfEnabled.Store(false)
		if perfFile != nil {
			perfFile.Close()
			perfFile = nil
		}
		return nil
	}
	if perfFile != nil {
		return nil
	}
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return err
	}
	file, err := os.Create(fmt.Sprintf("logs/perf_%s.jsonl", time.Now().Format("20060102_150405")))
	if err != nil {
		return err
	}
	perfFile = file
	perfEnabled.Store(true)
	return nil
}

func PerfWrite(entry map[string]any) {
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	perfMu.Lock()
	defer perfMu.Unlock()
	if perfStdout.Load() {
		fmt.Fprintf(os.Stdout, "PERF %s\n", line)
	} else if perfFile != nil {
		perfFile.Write(append(line, '\n'))
	}
}

// Fields is read at stop time, so callers may add to it before stopping
func PerfTimer(kind string, fields map[string]any) func() {
	if !PerfEnabled() {
		return func() {}
	}
	start := time.Now()
	return func() {
		entry := map[string]any{
			"source": "backend",
			"kind":   kind,
			"ts":     time.Now().UnixMilli(),
			"ms":     float64(time.Since(start).Microseconds()) / 1000,
		}
		maps.Copy(entry, fields)
		PerfWrite(entry)
	}
}
