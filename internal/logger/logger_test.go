package logger

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

func TestSimpleLogger_Info(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger := NewSimple()
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "INFO: test message") {
		t.Errorf("Expected log to contain 'INFO: test message', got: %s", output)
	}
}

func TestSimpleLogger_Error(t *testing.T) {
	// Capture stderr output
	var buf bytes.Buffer
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewSimple()
	testErr := errors.New("test error")
	logger.Error("test error message", testErr)

	w.Close()
	os.Stderr = oldStderr
	buf.ReadFrom(r)

	output := buf.String()
	if !strings.Contains(output, "ERROR: test error message: test error") {
		t.Errorf("Expected error log to contain error message, got: %s", output)
	}
}

func TestSimpleLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger := NewSimple()
	fieldLogger := logger.WithField("key", "value")
	fieldLogger.Info("test with field")

	output := buf.String()
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("Expected log to contain field key-value, got: %s", output)
	}
}

func TestSimpleLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger := NewSimple()
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("test with multiple fields")

	output := buf.String()
	if !strings.Contains(output, "key1") || !strings.Contains(output, "key2") {
		t.Errorf("Expected log to contain multiple fields, got: %s", output)
	}
}

func TestSimpleLogger_ChainedFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger := NewSimple()
	chainedLogger := logger.WithField("first", "value1").WithField("second", "value2")
	chainedLogger.Info("chained fields test")

	output := buf.String()
	if !strings.Contains(output, "first") || !strings.Contains(output, "second") {
		t.Errorf("Expected log to contain chained fields, got: %s", output)
	}
}