package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(environment string) *zap.Logger {
	var cfg zap.Config

	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get user config directory: %v\n", err)
		os.Exit(1)
	}

	logDir := filepath.Join(configDir, "ytmp", "logs")
	logFile := filepath.Join(logDir, "ytmp.log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	if environment == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.MessageKey = "message"
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.StacktraceKey = "stacktrace"
		cfg.DisableCaller = false
		cfg.DisableStacktrace = true
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		cfg.DisableStacktrace = true
	}

	cfg.OutputPaths = []string{logFile}
	cfg.ErrorOutputPaths = []string{logFile}

	logger, err := cfg.Build(
		zap.AddCallerSkip(0),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	return logger
}
