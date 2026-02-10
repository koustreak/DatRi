package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom json config",
			config: &Config{
				Level:  "debug",
				Format: "json",
			},
		},
		{
			name: "console config",
			config: &Config{
				Level:  "info",
				Format: "console",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			assert.NotNil(t, logger)
		})
	}
}

func TestLogger_JSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	logger.Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test message", logEntry["message"])
	assert.NotEmpty(t, logEntry["time"])
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	childLogger := logger.With().
		Str("service", "datri").
		Int("port", 8080).
		Logger()

	childLogger.Info("server started")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "datri", logEntry["service"])
	assert.Equal(t, float64(8080), logEntry["port"])
	assert.Equal(t, "server started", logEntry["message"])
}

func TestLogger_ErrorWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  "error",
		Format: "json",
		Output: buf,
	})

	testErr := errors.New("database connection failed")
	logger.ErrorWith("failed to connect", testErr, map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	})

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "error", logEntry["level"])
	assert.Equal(t, "failed to connect", logEntry["message"])
	assert.Equal(t, "database connection failed", logEntry["error"])
	assert.Equal(t, "localhost", logEntry["host"])
	assert.Equal(t, float64(5432), logEntry["port"])
}

func TestLogger_Context(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	ctx := logger.WithContext(context.Background())
	retrievedLogger := FromContext(ctx)

	retrievedLogger.Info("from context")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "from context", logEntry["message"])
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		logFunc  func(*Logger)
		expected bool // should log or not
	}{
		{
			name:  "debug level logs debug",
			level: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			expected: true,
		},
		{
			name:  "info level skips debug",
			level: "info",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			expected: false,
		},
		{
			name:  "error level logs error",
			level: "error",
			logFunc: func(l *Logger) {
				l.Error("error message")
			},
			expected: true,
		},
		{
			name:  "error level skips info",
			level: "error",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(&Config{
				Level:  tt.level,
				Format: "json",
				Output: buf,
			})

			tt.logFunc(logger)

			if tt.expected {
				assert.NotEmpty(t, buf.String(), "expected log output")
			} else {
				assert.Empty(t, buf.String(), "expected no log output")
			}
		})
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := New(&Config{
		Level:  "info",
		Format: "json",
		Output: io.Discard,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLogger_WithFields(b *testing.B) {
	logger := New(&Config{
		Level:  "info",
		Format: "json",
		Output: io.Discard,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.With().
			Str("service", "datri").
			Int("request_id", i).
			Logger().
			Info("benchmark message")
	}
}
