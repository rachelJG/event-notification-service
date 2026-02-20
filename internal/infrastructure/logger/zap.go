package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(env string, level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	if strings.EqualFold(env, "development") {
		cfg = zap.NewDevelopmentConfig()
	}

	var zapLevel zapcore.Level
	if err := zapLevel.Set(level); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	return cfg.Build()
}
