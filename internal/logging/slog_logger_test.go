package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewSlogLogger(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	slogLogger := NewSlogLogger(logger)

	if slogLogger == nil {
		t.Fatal("NewSlogLogger returned nil")
	}
	if slogLogger.logger != logger {
		t.Error("NewSlogLogger did not set logger correctly")
	}
}

func TestSlogLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := NewSlogLogger(slog.New(handler))

	logger.Debug("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Debug output missing message: %s", output)
	}
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Debug output missing level: %s", output)
	}
}

func TestSlogLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := NewSlogLogger(slog.New(handler))

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Info output missing message: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Info output missing level: %s", output)
	}
}

func TestSlogLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := NewSlogLogger(slog.New(handler))

	logger.Warn("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Warn output missing message: %s", output)
	}
	if !strings.Contains(output, "WARN") {
		t.Errorf("Warn output missing level: %s", output)
	}
}

func TestSlogLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	logger := NewSlogLogger(slog.New(handler))

	logger.Error("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Error output missing message: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Error output missing level: %s", output)
	}
}

func TestSlogLogger_With(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := NewSlogLogger(slog.New(handler))

	childLogger := logger.With("service", "test-service")
	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test-service") {
		t.Errorf("With() context not included in output: %s", output)
	}
}

func TestSlogLogger_IsLogLevelEnabled(t *testing.T) {
	tests := []struct {
		name          string
		handlerLevel  slog.Level
		checkLevel    string
		expectEnabled bool
	}{
		{"Debug enabled at Debug", slog.LevelDebug, "debug", true},
		{"Info enabled at Debug", slog.LevelDebug, "info", true},
		{"Debug disabled at Info", slog.LevelInfo, "debug", false},
		{"Info enabled at Info", slog.LevelInfo, "info", true},
		{"Warn enabled at Info", slog.LevelInfo, "warn", true},
		{"Error enabled at Info", slog.LevelInfo, "error", true},
		{"Info disabled at Error", slog.LevelError, "info", false},
		{"Error enabled at Error", slog.LevelError, "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: tt.handlerLevel})
			logger := NewSlogLogger(slog.New(handler))

			enabled := logger.IsLogLevelEnabled(tt.checkLevel)
			if enabled != tt.expectEnabled {
				t.Errorf("IsLogLevelEnabled(%q) = %v, want %v", tt.checkLevel, enabled, tt.expectEnabled)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"  debug  ", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // fallback
		{"unknown", slog.LevelInfo}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSlogLogger_KeyValuePairs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := NewSlogLogger(slog.New(handler))

	logger.Info("test message", "key1", "value1", "key2", 42, "key3", true)

	output := buf.String()

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("Message not found in output: %v", logEntry)
	}
	if logEntry["key1"] != "value1" {
		t.Errorf("key1 not found or incorrect: %v", logEntry)
	}
	if logEntry["key2"] != float64(42) { // JSON numbers are float64
		t.Errorf("key2 not found or incorrect: %v", logEntry)
	}
	if logEntry["key3"] != true {
		t.Errorf("key3 not found or incorrect: %v", logEntry)
	}
}
