package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

type SimpleLogger struct{}

func NewSimple() Logger {
	return &SimpleLogger{}
}

func (l *SimpleLogger) Info(msg string) {
	log.Printf("INFO: %s", msg)
}

func (l *SimpleLogger) Error(msg string, err error) {
	log.Printf("ERROR: %s: %v", msg, err)
}

func (l *SimpleLogger) WithField(key string, value interface{}) Logger {
	return l
}

func (l *SimpleLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}

type LogrusLogger struct{}

func NewLogrus() Logger {
	return &LogrusLogger{}
}

func (l *LogrusLogger) Info(msg string) {
	fmt.Fprintf(os.Stdout, "INFO: %s\n", msg)
}

func (l *LogrusLogger) Error(msg string, err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", msg, err)
}

func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return l
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}