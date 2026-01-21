// Package logger provides structured logging utilities
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelFatal LogLevel = "FATAL"
)

// Config holds logger configuration
type Config struct {
	Level      string `yaml:"level"`       // debug, info, warn, error, fatal
	Format     string `yaml:"format"`      // text or json
	Output     string `yaml:"output"`      // stdout, stderr, or file path
	TimeFormat string `yaml:"time_format"` // RFC3339, RFC3339Nano, etc
}

var (
	currentLevel      = LevelInfo
	currentFormat     = "text" // text or json
	currentTimeFormat = time.RFC3339
	infoLog           = log.New(os.Stdout, "", 0)
	errorLog          = log.New(os.Stderr, "", 0)
)

// Init initializes the logger with configuration
func Init(cfg Config) {
	// Set log level
	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	case "fatal":
		currentLevel = LevelFatal
	default:
		currentLevel = LevelInfo
	}

	// Set format
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "json":
		currentFormat = "json"
	default:
		currentFormat = "text"
	}

	// Set time format
	if strings.TrimSpace(cfg.TimeFormat) != "" {
		currentTimeFormat = strings.TrimSpace(cfg.TimeFormat)
	}

	// Set output
	switch strings.ToLower(strings.TrimSpace(cfg.Output)) {
	case "", "stdout":
		infoLog.SetOutput(os.Stdout)
		errorLog.SetOutput(os.Stderr)
	case "stderr":
		infoLog.SetOutput(os.Stderr)
		errorLog.SetOutput(os.Stderr)
	default:
		f, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			infoLog.SetOutput(os.Stdout)
			errorLog.SetOutput(os.Stderr)
			infoLog.Printf("logger: failed to open log file %s: %v", cfg.Output, err)
		} else {
			infoLog.SetOutput(f)
			errorLog.SetOutput(f)
		}
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Protocol  string                 `json:"protocol,omitempty"`
	File      string                 `json:"file,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// shouldLog checks if message should be logged based on level
func shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
	}
	return levels[level] >= levels[currentLevel]
}

// logMessage handles the actual logging
func logMessage(level LogLevel, msg string, fields map[string]interface{}) {
	if !shouldLog(level) {
		return
	}

	// Get caller info
	_, file, line, ok := runtime.Caller(2)
	if ok {
		parts := strings.Split(file, "/")
		file = fmt.Sprintf("%s:%d", parts[len(parts)-1], line)
	}

	var protocol string
	if fields != nil {
		if v, ok := fields["protocol"].(string); ok {
			protocol = v
		}
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(currentTimeFormat),
		Level:     string(level),
		Message:   msg,
		Protocol:  protocol,
		File:      file,
		Fields:    fields,
	}

	var output string
	if currentFormat == "json" {
		// JSON format
		data, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("%s [%s] %s", entry.Timestamp, entry.Level, entry.Message)
		} else {
			output = string(data)
		}
	} else {
		// Text format
		output = fmt.Sprintf("%s [%s] %s", entry.Timestamp, entry.Level, entry.Message)
		if entry.File != "" {
			output += fmt.Sprintf(" (%s)", entry.File)
		}
		if len(entry.Fields) > 0 {
			output += fmt.Sprintf(" %v", entry.Fields)
		}
	}

	// Route to appropriate output
	if level >= LevelError {
		errorLog.Println(output)
	} else {
		infoLog.Println(output)
	}

	// Fatal level terminates
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs debug message (only shown when level=debug)
func Debug(msg string) {
	logMessage(LevelDebug, msg, nil)
}

// Debugf logs formatted debug message
func Debugf(format string, args ...interface{}) {
	logMessage(LevelDebug, fmt.Sprintf(format, args...), nil)
}

// Info logs info message
func Info(msg string) {
	logMessage(LevelInfo, msg, nil)
}

// Infof logs formatted info message
func Infof(format string, args ...interface{}) {
	logMessage(LevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warn logs warning message
func Warn(msg string) {
	logMessage(LevelWarn, msg, nil)
}

// Warnf logs formatted warning message
func Warnf(format string, args ...interface{}) {
	logMessage(LevelWarn, fmt.Sprintf(format, args...), nil)
}

// Error logs error message
func Error(msg string) {
	logMessage(LevelError, msg, nil)
}

// Errorf logs formatted error message
func Errorf(format string, args ...interface{}) {
	logMessage(LevelError, fmt.Sprintf(format, args...), nil)
}

// Fatal logs fatal message and exits
func Fatal(msg string) {
	logMessage(LevelFatal, msg, nil)
}

// Fatalf logs formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	logMessage(LevelFatal, fmt.Sprintf(format, args...), nil)
}

// WithFields returns a log message with structured fields
func WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{fields: fields}
}

// FieldLogger allows structured logging with fields
type FieldLogger struct {
	fields map[string]interface{}
}

func (l *FieldLogger) Info(msg string) {
	logMessage(LevelInfo, msg, l.fields)
}

func (l *FieldLogger) Warn(msg string) {
	logMessage(LevelWarn, msg, l.fields)
}

func (l *FieldLogger) Error(msg string) {
	logMessage(LevelError, msg, l.fields)
}

// Protocol-specific logging with structured fields

// HTTP logs HTTP protocol activity
func HTTP(method, path string, status, latencyMs int) {
	WithFields(map[string]interface{}{
		"protocol": "http",
		"method":   method,
		"path":     path,
		"status":   status,
		"latency":  latencyMs,
	}).Info(fmt.Sprintf("HTTP %s %s %d - %dms", method, path, status, latencyMs))
}

// GRPC logs gRPC protocol activity
func GRPC(method, params string, latencyMs int) {
	WithFields(map[string]interface{}{
		"protocol": "grpc",
		"method":   method,
		"params":   params,
		"latency":  latencyMs,
	}).Info(fmt.Sprintf("gRPC %s(%s) - %dms", method, params, latencyMs))
}

// WebSocket logs WebSocket activity
func WebSocket(room, event string, userID string) {
	WithFields(map[string]interface{}{
		"protocol": "websocket",
		"room":     room,
		"event":    event,
		"user_id":  userID,
	}).Info(fmt.Sprintf("WebSocket [%s] %s", room, event))
}

// TCP logs TCP protocol activity
func TCP(eventType, mangaID string, count int) {
	WithFields(map[string]interface{}{
		"protocol":   "tcp",
		"event_type": eventType,
		"manga_id":   mangaID,
		"count":      count,
	}).Info(fmt.Sprintf("TCP %s (manga:%s, count:%d)", eventType, mangaID, count))
}

// UDP logs UDP broadcast activity
func UDP(notificationType, message string, recipientCount int) {
	WithFields(map[string]interface{}{
		"protocol":   "udp",
		"type":       notificationType,
		"recipients": recipientCount,
	}).Info(fmt.Sprintf("UDP broadcast %s to %d recipients", notificationType, recipientCount))
}

// Context-aware logging (for request tracing)
type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID extracts request ID from context and logs with it
func WithRequestID(ctx context.Context) *FieldLogger {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return WithFields(map[string]interface{}{
			"request_id": requestID,
		})
	}
	return WithFields(nil)
}
