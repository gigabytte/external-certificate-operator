package log

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestFromContext(t *testing.T) {
	ctx := context.TODO()
	logger := FromContext(ctx)
	assert.NotNil(t, logger)

	// Test with logger in context
	ctxWithLogger := NewContext(ctx, logger)
	loggerFromCtx := FromContext(ctxWithLogger)
	assert.Equal(t, logger, loggerFromCtx)
}

func TestNewContext(t *testing.T) {
	ctx := context.TODO()
	logger := FromContext(ctx)
	newCtx := NewContext(ctx, logger)
	assert.NotNil(t, newCtx)
}

func TestInfo(t *testing.T) {
	// Capture the log output
	logOutput := captureLogOutput(func() {
		Info("This is an info message", "key1", "value1")
	})
	assert.Contains(t, logOutput, "This is an info message")
	assert.Contains(t, logOutput, "key1")
	assert.Contains(t, logOutput, "value1")
}

func TestIntoContext(t *testing.T) {
	ctx := context.TODO()
	logger := FromContext(ctx)
	newCtx := IntoContext(ctx, logger)
	assert.NotNil(t, newCtx)
	loggerFromCtx := FromContext(newCtx)
	assert.Equal(t, logger, loggerFromCtx)
}

// Helper function to capture log output
func captureLogOutput(f func()) string {
	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)
	logrLogger = zapr.NewLogger(logger)
	f()
	return buf.String()
}
