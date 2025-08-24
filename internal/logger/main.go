package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	L           *zap.SugaredLogger
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
)

func Init() {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		panic(err)
	}
	logger, err := newZapLogger()
	if err != nil {
		panic(err)
	}
	L = logger.Sugar()
}

func SetLevel(level zapcore.Level) {
	atomicLevel.SetLevel(level)
}
