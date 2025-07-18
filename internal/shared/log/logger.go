package log

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// Logger is the global logger instance
	Logger = newLogger()
)

// newLogger sets up a new logger with JSON encoding
func newLogger() logr.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config.Encoding = "json"
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// Set the logger as the default logger for controller-runtime
	log.SetLogger(zapr.NewLogger(zapLogger))

	return zapr.NewLogger(zapLogger)
}

// FromContext retrieves the logger from the context or returns the default logger.
func FromContext(ctx context.Context) logr.Logger {
	if ctxLogger, err := logr.FromContext(ctx); err == nil {
		return ctxLogger
	}
	return Logger
}

// WithName returns a logger with the specified name
func WithName(name string) logr.Logger {
	return Logger.WithName(name)
}

// WithContext returns a logger with the provided context
func WithContext(ctx context.Context) logr.Logger {
	return FromContext(ctx)
}

// WithValues returns a logger with the provided key-value pairs
func WithValues(keysAndValues ...interface{}) logr.Logger {
	return Logger.WithValues(keysAndValues...)
}

// IntoContext adds the logger to the provided context
func IntoContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}
