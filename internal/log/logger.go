package log

import (
	"context"
	"log"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

var logger *zap.Logger
var logrLogger logr.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}

	logrLogger = zapr.NewLogger(logger)

	// Redirect klog to zap
	klog.SetLogger(logrLogger)

	// Redirect standard library log output to zap
	log.SetOutput(zapWriter{logger: logger, level: zapcore.InfoLevel})
}

// FromContext retrieves the logger from the context or returns the default logger.
func FromContext(ctx context.Context) logr.Logger {
	if ctxLogger, err := logr.FromContext(ctx); err == nil {
		return ctxLogger
	}
	return logrLogger
}

// NewContext returns a new context with the provided logger.
func NewContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}

// Info logs an informational message with optional key-value pairs.
func Info(msg string, keysAndValues ...interface{}) {
	logrLogger.Info(msg, keysAndValues...)
}

// WithName returns a new logger with the specified name.
func WithName(name string) logr.Logger {
	return logrLogger.WithName(name)
}

// IntoContext takes a context and sets the logger as one of its values.
// Use FromContext function to retrieve the logger.
func IntoContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}

type zapWriter struct {
	logger *zap.Logger
	level  zapcore.Level
}

func (z zapWriter) Write(p []byte) (n int, err error) {
	z.logger.Check(z.level, string(p)).Write()
	return len(p), nil
}
