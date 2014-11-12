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

type SimpleLogger struct{}

func (l *SimpleLogger) Info(args ...interface{}) {
	data := append([]interface{}{"INFO"}, args...)
	log.Print(data...)
}

func (l *SimpleLogger) Warn(args ...interface{}) {
	data := append([]interface{}{"WARN"}, args...)
	log.Print(data...)
}

func (l *SimpleLogger) Error(args ...interface{}) {
	data := append([]interface{}{"ERROR"}, args...)
	log.Print(data...)
}

func (l *SimpleLogger) Fatal(args ...interface{}) {
	data := append([]interface{}{"FATAL"}, args...)
	log.Fatal(data...)
}

func (l *SimpleLogger) Print(args ...interface{}) {
	log.Print(args...)
}
