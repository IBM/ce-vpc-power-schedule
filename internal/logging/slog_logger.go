package logging

import (
	"context"
	"log/slog"
	"strings"
)

type SlogLogger struct{ logger *slog.Logger }

func NewSlogLogger(logger *slog.Logger) *SlogLogger { return &SlogLogger{logger: logger} }

func (slog *SlogLogger) Debug(msg string, kv ...any) { slog.logger.Debug(msg, kv...) }
func (slog *SlogLogger) Info(msg string, kv ...any)  { slog.logger.Info(msg, kv...) }
func (slog *SlogLogger) Warn(msg string, kv ...any)  { slog.logger.Warn(msg, kv...) }
func (slog *SlogLogger) Error(msg string, kv ...any) { slog.logger.Error(msg, kv...) }
func (slog *SlogLogger) With(kv ...any) Logger       { return &SlogLogger{logger: slog.logger.With(kv...)} }

func (slog *SlogLogger) IsLogLevelEnabled(level string) bool {
	logLevel := parseLevel(level)
	return slog.logger.Enabled(context.Background(), logLevel)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Fallback: treat unknown as info
		return slog.LevelInfo
	}
}
