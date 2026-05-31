package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

type Logger struct {
	service string
	level   Level
	out     io.Writer
}

type LogEntry struct {
	Timestamp string                 `json:"ts"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"msg"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func New(service string) *Logger {
	lvl := LevelInfo
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			lvl = LevelDebug
		case "warn":
			lvl = LevelWarn
		case "error":
			lvl = LevelError
		}
	}
	return &Logger{service: service, level: lvl, out: os.Stdout}
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}
	_, file, line, _ := runtime.Caller(2)
	short := file
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		short = file[idx+1:]
	}
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Service:   l.service,
		Message:   msg,
		Caller:    fmt.Sprintf("%s:%d", short, line),
		Fields:    fields,
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(l.out, string(data))
}

func (l *Logger) Info(msg string, kv ...interface{})  { l.log(LevelInfo, msg, kvToMap(kv)) }
func (l *Logger) Warn(msg string, kv ...interface{})  { l.log(LevelWarn, msg, kvToMap(kv)) }
func (l *Logger) Error(msg string, kv ...interface{}) { l.log(LevelError, msg, kvToMap(kv)) }
func (l *Logger) Debug(msg string, kv ...interface{}) { l.log(LevelDebug, msg, kvToMap(kv)) }

func kvToMap(kv []interface{}) map[string]interface{} {
	if len(kv) == 0 {
		return nil
	}
	m := make(map[string]interface{})
	for i := 0; i < len(kv)-1; i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprintf("key_%d", i)
		}
		m[key] = kv[i+1]
	}
	return m
}
