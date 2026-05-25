package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Field struct {
	Key   string
	Value any
}

type Logger interface {
	With(fields ...Field) Logger
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

type slogLogger struct {
	inner *slog.Logger
}

func New(level string) Logger {
	var slogLevel slog.Level

	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})

	return slogLogger{inner: slog.New(handler)}
}

func Nop() Logger {
	handler := slog.NewJSONHandler(io.Discard, nil)
	return slogLogger{inner: slog.New(handler)}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

func (l slogLogger) With(fields ...Field) Logger {
	return slogLogger{inner: l.inner.With(toArgs(fields)...)}
}

func (l slogLogger) Debug(msg string, fields ...Field) {
	l.inner.Debug(msg, toArgs(fields)...)
}

func (l slogLogger) Info(msg string, fields ...Field) {
	l.inner.Info(msg, toArgs(fields)...)
}

func (l slogLogger) Warn(msg string, fields ...Field) {
	l.inner.Warn(msg, toArgs(fields)...)
}

func (l slogLogger) Error(msg string, fields ...Field) {
	l.inner.Error(msg, toArgs(fields)...)
}

func toArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	return args
}
