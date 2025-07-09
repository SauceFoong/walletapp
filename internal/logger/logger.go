package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init initializes the logger with proper configuration
func Init() {
	log = logrus.New()

	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // default level
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output to stdout
	log.SetOutput(os.Stdout)

	// Set JSON formatter for structured logging
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
}

// Get returns the configured logger instance
func Get() *logrus.Logger {
	if log == nil {
		Init()
	}
	return log
}

// WithField creates a new logger with a field
func WithField(key string, value interface{}) *logrus.Entry {
	return Get().WithField(key, value)
}

// WithFields creates a new logger with multiple fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

// Convenience methods for common logging patterns
func Info(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Info(msg)
	} else {
		Get().Info(msg)
	}
}

func Error(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Error(msg)
	} else {
		Get().Error(msg)
	}
}

func Warn(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Warn(msg)
	} else {
		Get().Warn(msg)
	}
}

func Debug(msg string, fields ...logrus.Fields) {
	if len(fields) > 0 {
		Get().WithFields(fields[0]).Debug(msg)
	} else {
		Get().Debug(msg)
	}
}

// WithUser creates a logger with user context
func WithUser(userID string) *logrus.Entry {
	return Get().WithField("user_id", userID)
}

// WithTransaction creates a logger with transaction context
func WithTransaction(txID string) *logrus.Entry {
	return Get().WithField("transaction_id", txID)
}

// WithOperation creates a logger with operation context
func WithOperation(operation string) *logrus.Entry {
	return Get().WithField("operation", operation)
}
