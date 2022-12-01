package log

import (
	"github.com/tryfix/log"
)

type Logger struct {
	logEnabled bool
	log.Logger
}

func NewLogger(logEnabled bool) *Logger {
	return &Logger{
		logEnabled: logEnabled,
		Logger: log.Constructor.Log(
			log.WithColors(true),
			log.WithLevel("TRACE"),
			log.WithFilePath(true),
			log.WithSkipFrameCount(4), // todo fix this count for different logs
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
