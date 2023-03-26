package log

import (
	"github.com/tryfix/log"
)

const (
	LevelTrace = "TRACE"
	LevelError = "ERROR"
)

type Logger struct {
	logEnabled bool
	log.Logger
}

func NewLogger(logEnabled bool, skipCount int, level string) *Logger {
	return &Logger{
		logEnabled: logEnabled,
		Logger: log.Constructor.Log(
			log.WithColors(true),
			log.WithLevel(log.Level(level)),
			log.WithFilePath(true),
			log.WithSkipFrameCount(skipCount),
		),
	}
}

func (l *Logger) Error(message interface{}, params ...interface{}) {
	if l.logEnabled {
		l.Logger.Error(message, params)
	}
}

func (l *Logger) Trace(message interface{}, params ...interface{}) {
	if l.logEnabled {
		l.Logger.Trace(message, params)
	}
}

func (l *Logger) Debug(message interface{}, params ...interface{}) {
	if l.logEnabled {
		l.Logger.Debug(message, params)
	}
}

func (l *Logger) Info(message interface{}, params ...interface{}) {
	if l.logEnabled {
		l.Logger.Info(message, params)
	}
}
