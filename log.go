package irc

import (
	"log"
)

// Simple logger interface designed for use with logrous
// and other similar systems
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Print(args ...interface{})
}

type DefaultLogger struct{}

func (l *DefaultLogger) Info(args ...interface{}) {
	data := append([]interface{}{"INFO"}, args...)
	log.Print(data...)
}

func (l *DefaultLogger) Warn(args ...interface{}) {
	data := append([]interface{}{"WARN"}, args...)
	log.Print(data...)
}

func (l *DefaultLogger) Error(args ...interface{}) {
	data := append([]interface{}{"ERROR"}, args...)
	log.Print(data...)
}

func (l *DefaultLogger) Fatal(args ...interface{}) {
	data := append([]interface{}{"FATAL"}, args...)
	log.Fatal(data...)
}

func (l *DefaultLogger) Print(args ...interface{}) {
	log.Print(args...)
}
