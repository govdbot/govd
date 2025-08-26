package database

import "go.uber.org/zap"

type gooseLogger struct {
	log *zap.SugaredLogger
}

func (l gooseLogger) Fatalf(format string, v ...interface{}) {
	l.log.Fatalf(format, v...)
}

func (l gooseLogger) Printf(format string, v ...interface{}) {
	l.log.Infof(format, v...)
}
