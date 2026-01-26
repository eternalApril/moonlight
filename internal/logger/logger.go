package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a configured logger
// level: "debug", "info", "warn", "error"
// encoding: "json" (production) or "console" (development)
func New(level string, encoding string) *zap.Logger {
	// Parse level
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(lvl),
		Development: encoding == "console",
		Encoding:    encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := cfg.Build()
	if err != nil {
		// if logger fails, fallback to basic stdout and exit
		os.Stdout.WriteString("FAILED TO INIT LOGGER: " + err.Error())
		os.Exit(1)
	}

	return logger
}
