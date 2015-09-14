package irc

import (
	"log"
)

// Logger is a simple logger interface designed for use with logrus
// and other similar systems
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Print(args ...interface{})
}

// NilLogger AKA Black hole logger or /dev/null logger.
type NilLogger struct{}

// Debug implements Logger.Debug
func (l *NilLogger) Debug(args ...interface{}) {}

// Info implements Logger.Info
func (l *NilLogger) Info(args ...interface{}) {}

// Warn implements Logger.Warn
func (l *NilLogger) Warn(args ...interface{}) {}

// Error implements Logger.Error
func (l *NilLogger) Error(args ...interface{}) {}

// Fatal implements Logger.Fatal
func (l *NilLogger) Fatal(args ...interface{}) {}

// Print implements Logger.Print
func (l *NilLogger) Print(args ...interface{}) {}

// SimpleLogger simply tries to proxy to the official golang log package
type SimpleLogger struct{}

// Debug implements Logger.Debug
func (l *SimpleLogger) Debug(args ...interface{}) {
	data := append([]interface{}{"DEBUG"}, args...)
	log.Print(data...)
}

// Info implements Logger.Info
func (l *SimpleLogger) Info(args ...interface{}) {
	data := append([]interface{}{"INFO"}, args...)
	log.Print(data...)
}

// Warn implements Logger.Warn
func (l *SimpleLogger) Warn(args ...interface{}) {
	data := append([]interface{}{"WARN"}, args...)
	log.Print(data...)
}

// Error implements Logger.Error
func (l *SimpleLogger) Error(args ...interface{}) {
	data := append([]interface{}{"ERROR"}, args...)
	log.Print(data...)
}

// Fatal implements Logger.Fatal
func (l *SimpleLogger) Fatal(args ...interface{}) {
	data := append([]interface{}{"FATAL"}, args...)
	log.Fatal(data...)
}

// Print implements Logger.Print
func (l *SimpleLogger) Print(args ...interface{}) {
	log.Print(args...)
}
