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

func NewSimple() Logger {
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

type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

func NewLogrus() Logger {
	logger := logrus.New()
	return &LogrusLogger{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

func (l *LogrusLogger) Info(msg string) {
	l.entry.Info(msg)
}

func (l *LogrusLogger) Error(msg string, err error) {
	l.entry.WithError(err).Error(msg)
}

func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}
