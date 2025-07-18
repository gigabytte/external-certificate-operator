package log

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(logr.Logger)
		expectedLevel zapcore.Level
		expectedMsg   string
		expectedKeys  []string
	}{
		{
			name: "Info log",
			logFunc: func(logger logr.Logger) {
				logger.Info("test message", "key1", "value1", "key2", "value2")
			},
			expectedLevel: zapcore.InfoLevel,
			expectedMsg:   "test message",
			expectedKeys:  []string{"key1", "key2"},
		},
		{
			name: "Error log",
			logFunc: func(logger logr.Logger) {
				logger.Error(nil, "error message", "key1", "value1")
			},
			expectedLevel: zapcore.ErrorLevel,
			expectedMsg:   "error message",
			expectedKeys:  []string{"key1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.InfoLevel)
			logger := zapr.NewLogger(zap.New(core))

			tt.logFunc(logger)

			logs := recorded.All()
			if len(logs) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(logs))
			}

			logEntry := logs[0]
			if logEntry.Level != tt.expectedLevel {
				t.Errorf("expected level %v, got %v", tt.expectedLevel, logEntry.Level)
			}
			if logEntry.Message != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, logEntry.Message)
			}

			fields := logEntry.ContextMap()
			for _, key := range tt.expectedKeys {
				if _, ok := fields[key]; !ok {
					t.Errorf("expected key %q in log fields", key)
				}
			}
		})
	}
}

func TestLoggerFunctions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "FromContext",
			testFunc: func() error {
				core, _ := observer.New(zapcore.InfoLevel)
				logger := zapr.NewLogger(zap.New(core))

				ctx := IntoContext(context.Background(), logger)
				retrievedLogger := FromContext(ctx)
				if retrievedLogger != logger {
					return fmt.Errorf("expected logger from context, got different logger")
				}
				return nil
			},
		},
		{
			name: "WithName",
			testFunc: func() error {
				core, recorded := observer.New(zapcore.InfoLevel)
				logger := zapr.NewLogger(zap.New(core))

				namedLogger := logger.WithName("testLogger")
				namedLogger.Info("test message")
				logs := recorded.All()
				if len(logs) != 1 {
					return fmt.Errorf("expected 1 log entry, got %d", len(logs))
				}
				if logs[0].LoggerName != "testLogger" {
					return fmt.Errorf("expected logger name 'testLogger', got %s", logs[0].LoggerName)
				}
				return nil
			},
		},
		{
			name: "WithContext",
			testFunc: func() error {
				core, _ := observer.New(zapcore.InfoLevel)
				logger := zapr.NewLogger(zap.New(core))

				ctx := context.Background()
				loggerWithCtx := WithContext(ctx)
				if loggerWithCtx != Logger {
					return fmt.Errorf("expected default logger, got different logger")
				}

				ctx = IntoContext(ctx, logger)
				loggerWithCtx = WithContext(ctx)
				if loggerWithCtx != logger {
					return fmt.Errorf("expected logger from context, got different logger")
				}
				return nil
			},
		},
		{
			name: "WithValues",
			testFunc: func() error {
				core, recorded := observer.New(zapcore.InfoLevel)
				logger := zapr.NewLogger(zap.New(core))

				loggerWithValues := logger.WithValues("key1", "value1")
				loggerWithValues.Info("test message")
				logs := recorded.All()
				if len(logs) != 1 {
					return fmt.Errorf("expected 1 log entry, got %d", len(logs))
				}

				fields := logs[0].ContextMap()
				if fields["key1"] != "value1" {
					return fmt.Errorf("expected key 'key1' with value 'value1', got %v", fields["key1"])
				}
				return nil
			},
		},
		{
			name: "IntoContext",
			testFunc: func() error {
				core, _ := observer.New(zapcore.InfoLevel)
				logger := zapr.NewLogger(zap.New(core))

				ctx := IntoContext(context.Background(), logger)
				retrievedLogger := FromContext(ctx)
				if retrievedLogger != logger {
					return fmt.Errorf("expected logger from context, got different logger")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.testFunc(); err != nil {
				t.Error(err)
			}
		})
	}
}
