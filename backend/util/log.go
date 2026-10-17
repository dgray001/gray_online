package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	out   io.Writer
	level LogLevel
	mu    sync.Mutex
}

func NewLogger(out io.Writer, level LogLevel) *Logger {
	if out == nil {
		out = io.Discard
	}
	return &Logger{out: out, level: level}
}

var (
	DefaultStd = NewLogger(os.Stdout, LevelInfo)
	DefaultErr = NewLogger(os.Stderr, LevelWarn)
)

func levelTag(lvl LogLevel) string {
	switch lvl {
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

func (l *Logger) Writer() io.Writer {
	if l == nil {
		return io.Discard
	}
	return l.out
}

func (l *Logger) SetOutput(w io.Writer) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if w == nil {
		w = io.Discard
	}
	l.out = w
}

func (l *Logger) SetLevel(level LogLevel) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) Level() LogLevel {
	if l == nil {
		return LevelInfo
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func IsDebug(l *Logger) bool {
	return l != nil && l.level <= LevelDebug && l.out != io.Discard
}

func Logf(l *Logger, level LogLevel, format string, args ...any) {
	if l == nil || l.level > level || l.out == io.Discard {
		return
	}
	msg := strings.TrimRight(fmt.Sprintf(format, args...), "\n")
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("20060102-15:04:05.000"), levelTag(level), msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	io.WriteString(l.out, line)
}

func Log(l *Logger, level LogLevel, args ...any) {
	if l == nil || l.level > level || l.out == io.Discard {
		return
	}
	msg := strings.TrimRight(fmt.Sprintln(args...), "\n")
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("20060102-15:04:05.000"), levelTag(level), msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	io.WriteString(l.out, line)
}

func Debugf(l *Logger, format string, args ...any) { Logf(l, LevelDebug, format, args...) }
func Debug(l *Logger, args ...any)                 { Log(l, LevelDebug, args...) }
func Infof(l *Logger, format string, args ...any)  { Logf(l, LevelInfo, format, args...) }
func Info(l *Logger, args ...any)                  { Log(l, LevelInfo, args...) }
func Warnf(l *Logger, format string, args ...any)  { Logf(l, LevelWarn, format, args...) }
func Warn(l *Logger, args ...any)                  { Log(l, LevelWarn, args...) }
func Errorf(l *Logger, format string, args ...any) { Logf(l, LevelError, format, args...) }
func Error(l *Logger, args ...any)                 { Log(l, LevelError, args...) }

// RedirectStderr pipes os.Stderr through a tee that prefixes each line "ERROR: " and writes
// it to every destination given (nil entries are skipped). Callers must invoke the returned
// cleanup func before exit to flush and restore ordering.
func RedirectStderr(destinations ...*os.File) (func(), error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	origStderr := os.Stderr
	origDefaultErrOut := DefaultErr.Writer()
	os.Stderr = write
	DefaultErr.SetOutput(write)
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(read)
		for scanner.Scan() {
			txt := strings.TrimRight(scanner.Text(), "\n")
			line := fmt.Sprintf("%s [ERROR] %s\n", time.Now().Format("20060102-15:04:05.000"), txt)
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
		os.Stderr = origStderr
		DefaultErr.SetOutput(origDefaultErrOut)
	}, nil
}
