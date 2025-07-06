package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

type SimpleLogger struct {
	fields map[string]interface{}
}

func NewSimpleLogger() Logger {
	return &SimpleLogger{
		fields: make(map[string]interface{}),
	}
}

func (l *SimpleLogger) Info(msg string) {
	if len(l.fields) > 0 {
		log.Printf("INFO: %s %v", msg, l.fields)
	} else {
		log.Printf("INFO: %s", msg)
	}
}

func (l *SimpleLogger) Error(msg string, err error) {
	if len(l.fields) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %v %v\n", msg, err, l.fields)
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", msg, err)
	}
}

func (l *SimpleLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	return &SimpleLogger{fields: newFields}
}

func (l *SimpleLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &SimpleLogger{fields: newFields}
}

// LogrusLogger implements the Logger interface using logrus
type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrus creates a new logrus-based logger
func NewLogrus() *LogrusLogger {
	logger := logrus.New()
	return &LogrusLogger{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// Info logs an info message
func (l *LogrusLogger) Info(msg string) {
	l.entry.Info(msg)
}

// Error logs an error message
func (l *LogrusLogger) Error(msg string, err error) {
	l.entry.WithError(err).Error(msg)
}

// WithField returns a logger with a field
func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields returns a logger with fields
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}

// SetLevel sets the log level
func (l *LogrusLogger) SetLevel(level logrus.Level) {
	l.logger.SetLevel(level)
}

// SetFormatter sets the log formatter
func (l *LogrusLogger) SetFormatter(formatter logrus.Formatter) {
	l.logger.SetFormatter(formatter)
}