package colored_logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/term"
)

type FdWriter interface {
	io.Writer
	Fd() uintptr
}

type Logger struct {
	depth     int
	mu        sync.RWMutex
	color     bool
	out       FdWriter
	debug     bool
	timestamp bool
	quiet     bool
	logFile   *os.File
	buf       ColorBuffer
}

// TODO: Singleton methods
var L *Logger

type Prefix struct {
	Plain     []byte
	Color     []byte
	File      bool
	Callstack bool
}

var (
	plainFatal = []byte("[FATAL] ")
	plainError = []byte("[ERROR] ")
	plainWarn  = []byte("[WARN]  ")
	plainInfo  = []byte("[INFO]  ")
	plainDebug = []byte("[DEBUG] ")
	plainTrace = []byte("[TRACE] ")

	FatalPrefix = Prefix{
		Plain: plainFatal,
		Color: Red(plainFatal),
		File:  true,
	}
	ErrorPrefix = Prefix{
		Plain: plainError,
		Color: Red(plainError),
		File:  true,
	}
	WarnPrefix = Prefix{
		Plain: plainWarn,
		Color: Orange(plainWarn),
	}
	InfoPrefix = Prefix{
		Plain: plainInfo,
		Color: Green(plainInfo),
	}
	DebugPrefix = Prefix{
		Plain: plainDebug,
		Color: Purple(plainDebug),
		File:  true,
	}
	TracePrefix = Prefix{
		Plain:     plainTrace,
		Color:     Cyan(plainTrace),
		Callstack: true,
	}
)

// New returns new Logger instance with predefined writer output and
// automatically detect terminal coloring support
func New(out FdWriter) *Logger {
	return &Logger{
		color:     term.IsTerminal(int(out.Fd())),
		out:       out,
		timestamp: true,
	}
}

// WithColor explicitly turns on colorful features on the logger
func (l *Logger) WithColor() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color = true
	return l
}

// Sets the depth in reflection for debug logs
func (l *Logger) Depth(depth int) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.depth = depth
	return l
}

func (l *Logger) IsColored() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.color
}

// WithoutColor explicitly turns off colorful features on the log
func (l *Logger) WithoutColor() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color = false
	return l
}

// WithLogFile turns on log saving to log file
func (l *Logger) WithLogFile(path string) *Logger {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		l.Error("could not open log file", path, err)
		return l
	}
	runtime.SetFinalizer(l, func(l *Logger) {
		l.Info("closing log file")
		l.logFile.Close()
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logFile = f
	return l
}

// WithDebug turns on debugging output on the log to reveal debug and trace level
func (l *Logger) WithDebug() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debug = true
	return l
}

func (l *Logger) WithoutDebug() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debug = false
	return l
}

func (l *Logger) IsDebug() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.debug
}

// WithTimestamp turns on timestamp output on the log
func (l *Logger) WithTimestamp() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timestamp = true
	return l
}

func (l *Logger) WithoutTimestamp() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timestamp = false
	return l
}

// Quiet turns off all log output
func (l *Logger) Quiet() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quiet = true
	return l
}

func (l *Logger) NoQuiet() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quiet = false
	return l
}

func (l *Logger) IsQuiet() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.quiet
}

func (l *Logger) Output(prefix Prefix, data string) error {
	if l.logFile != nil {
		loggerCopy := *l
		// must use a new mutex to avoid dead locks or concurrent writes to l.mu
		loggerCopy.mu = sync.RWMutex{}
		loggerCopy.color = false
		loggerCopy.out = loggerCopy.logFile
		if err := loggerCopy.output(prefix, data); err != nil {
			return err
		}
	}
	return l.output(prefix, data)
}

func (l *Logger) output(prefix Prefix, data string) error {
	if l.IsQuiet() {
		return nil
	}
	now := time.Now()
	// Acquire exclusive access to the shared buffer
	l.mu.Lock()
	defer l.mu.Unlock()
	// Reset buffer so it start from the begining
	l.buf.Reset()
	// Write prefix to the buffer
	if l.color {
		l.buf.Append(prefix.Color)
	} else {
		l.buf.Append(prefix.Plain)
	}
	if l.timestamp {
		if l.color {
			l.buf.Blue()
		}
		year, month, day := now.Date()
		l.buf.AppendInt(year, 4)
		l.buf.AppendByte('/')
		l.buf.AppendInt(int(month), 2)
		l.buf.AppendByte('/')
		l.buf.AppendInt(day, 2)
		l.buf.AppendByte(' ')
		hour, min, sec := now.Clock()
		l.buf.AppendInt(hour, 2)
		l.buf.AppendByte(':')
		l.buf.AppendInt(min, 2)
		l.buf.AppendByte(':')
		l.buf.AppendInt(sec, 2)
		l.buf.AppendByte(' ')
		// Print reset color if color enabled
		if l.color {
			l.buf.Off()
		}
	}
	// Add caller filename and line if enabled
	if prefix.File {
		file, fn, line, _ := l.getOccurrence(0)
		if l.color {
			l.buf.Orange()
		}
		// Print filename and line
		l.buf.Append([]byte(fn))
		l.buf.AppendByte(':')
		l.buf.Append([]byte(file))
		l.buf.AppendByte(':')
		l.buf.AppendInt(line, 0)
		l.buf.AppendByte(' ')
		// Print color stop
		if l.color {
			l.buf.Off()
		}
	}
	// Print the actual string data from caller
	l.buf.Append([]byte(data))
	if len(data) == 0 || data[len(data)-1] != '\n' {
		l.buf.AppendByte('\n')
	}
	// add call stack trace if enabled
	if prefix.Callstack {
		var ok bool
		maxCallDepth := 50
		for i := range maxCallDepth {
			var file, fn string
			var line int
			file, fn, line, ok = l.getOccurrence(i)
			if !ok {
				break
			}
			if l.color {
				l.buf.Gray()
			}
			// Print filename and line
			l.buf.AppendByte('\t')
			l.buf.Append([]byte(fn))
			l.buf.AppendByte(':')
			l.buf.Append([]byte(file))
			l.buf.AppendByte(':')
			l.buf.AppendInt(line, 0)
			l.buf.AppendByte('\n')
			// Print color stop
			if l.color {
				l.buf.Off()
			}
		}
	}
	// Flush buffer to output
	_, err := l.out.Write(l.buf.Buffer)
	return err
}

func (l *Logger) getOccurrence(additionalDepth int) (file, fn string, line int, ok bool) {
	var pc uintptr

	// Get the caller filename and line
	if pc, file, line, ok = runtime.Caller(l.depth + 2 + additionalDepth); !ok {
		file = "<unknown file>"
		fn = "<unknown function>"
		line = 0
	} else {
		file = filepath.Base(file)
		fn = runtime.FuncForPC(pc).Name()
	}
	return
}

// Fatal print fatal message to output and quit the application with status 1
func (l *Logger) Fatal(v ...interface{}) {
	l.Output(FatalPrefix, fmt.Sprintln(v...))
	os.Exit(1)
}

// Fatalf print formatted fatal message to output and quit the application
// with status 1
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(FatalPrefix, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Error print error message to output
func (l *Logger) Error(v ...interface{}) {
	l.Output(ErrorPrefix, fmt.Sprintln(v...))
}

// Errorf print formatted error message to output
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Output(ErrorPrefix, fmt.Sprintf(format, v...))
}

// Warn print warning message to output
func (l *Logger) Warn(v ...interface{}) {
	l.Output(WarnPrefix, fmt.Sprintln(v...))
}

// Warnf print formatted warning message to output
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Output(WarnPrefix, fmt.Sprintf(format, v...))
}

// Info print informational message to output
func (l *Logger) Info(v ...interface{}) {
	l.Output(InfoPrefix, fmt.Sprintln(v...))
}

// Infof print formatted informational message to output
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Output(InfoPrefix, fmt.Sprintf(format, v...))
}

// Debug print debug message to output if debug output enabled
func (l *Logger) Debug(v ...interface{}) {
	if l.IsDebug() {
		l.Output(DebugPrefix, fmt.Sprintln(v...))
	}
}

// Debugf print formatted debug message to output if debug output enabled
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.IsDebug() {
		l.Output(DebugPrefix, fmt.Sprintf(format, v...))
	}
}

// Trace print trace message to output if debug output enabled
func (l *Logger) Trace(v ...interface{}) {
	if l.IsDebug() {
		l.Output(TracePrefix, fmt.Sprintln(v...))
	}
}

// Tracef print formatted trace message to output if debug output enabled
func (l *Logger) Tracef(format string, v ...interface{}) {
	l.Trace(fmt.Sprintf(format, v...))
}
