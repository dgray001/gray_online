package util

import (
	"bufio"
	"io"
	"log"
	"os"
)

// Discarded by default; callers can point this at a file for debug runs.
var DebugLog = log.New(io.Discard, "", log.LstdFlags)

// RedirectStderr pipes os.Stderr through a tee that prefixes each line "ERROR: " and writes
// it to every destination given (nil entries are skipped). Callers must invoke the returned
// cleanup func before exit to flush and restore ordering.
func RedirectStderr(destinations ...*os.File) (func(), error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stderr = write
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(read)
		for scanner.Scan() {
			line := "ERROR: " + scanner.Text() + "\n"
			for _, dest := range destinations {
				if dest != nil {
					dest.WriteString(line)
				}
			}
		}
	}()
	return func() {
		write.Close()
		<-done
	}, nil
}
