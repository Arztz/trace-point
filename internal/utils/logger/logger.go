package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents log severity levels
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// Logger provides structured logging functionality
type Logger struct {
	logger *log.Logger
	mu     sync.RWMutex
	level  Level
	output io.Writer
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(os.Stdout, InfoLevel)
	})
	return defaultLogger
}

// New creates a new logger with the given output and level
func New(output io.Writer, level Level) *Logger {
	return &Logger{
		logger: log.New(output, "", 0),
		level:  level,
		output: output,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
	l.output = w
}

// log writes a log entry with the given level and message
func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	l.logger.Printf("%s [%s] %s", timestamp, levelName, message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
	os.Exit(1)
}

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	Default().Debug(format, args...)
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	Default().Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	Default().Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	Default().Error(format, args...)
}

// Fatal logs a fatal message and exits using the default logger
func Fatal(format string, args ...interface{}) {
	Default().Fatal(format, args...)
}

// SetLevel sets the log level for the default logger
func SetLevel(level Level) {
	Default().SetLevel(level)
}

// ParseLevel converts a string to a Level
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}
