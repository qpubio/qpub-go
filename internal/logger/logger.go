package logger

import (
	"fmt"
	"log"
	"strings"

	"github.com/qpubio/qpub-go/option"
)

// Logger is a component-scoped logger.
type Logger struct {
	instanceID string
	component  string
	opts       option.Option
}

// Factory creates loggers.
type Factory struct {
	instanceID string
	opts       option.Option
}

func NewFactory(instanceID string, opts option.Option) *Factory {
	return &Factory{instanceID: instanceID, opts: opts}
}

func (f *Factory) Create(component string) *Logger {
	return &Logger{instanceID: f.instanceID, component: component, opts: f.opts}
}

func (l *Logger) levelRank() map[string]int {
	return map[string]int{"trace": 0, "debug": 1, "info": 2, "warn": 3, "error": 4}
}

func (l *Logger) shouldLog(level string) bool {
	if l.opts.Debug {
		return true
	}
	cur := l.levelRank()[strings.ToLower(l.opts.LogLevel)]
	lv := l.levelRank()[strings.ToLower(level)]
	return lv >= cur
}

func (l *Logger) log(level, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s][%s][%s] %s", level, l.instanceID, l.component, msg)
}

func (l *Logger) Error(format string, args ...interface{}) { l.log("error", format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log("warn", format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log("info", format, args...) }
func (l *Logger) Debug(format string, args ...interface{}) { l.log("debug", format, args...) }
func (l *Logger) Trace(format string, args ...interface{}) { l.log("trace", format, args...) }
