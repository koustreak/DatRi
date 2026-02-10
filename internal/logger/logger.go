package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog with enterprise-grade features
type Logger struct {
	zlog zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	TimeFormat string // rfc3339, unix, etc.
	Output     io.Writer
}

// DefaultConfig returns production-ready defaults
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		TimeFormat: "rfc3339",
		Output:     os.Stdout,
	}
}

// New creates a new enterprise logger
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set global log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = getTimeFormat(cfg.TimeFormat)

	// Create base logger
	var zlog zerolog.Logger
	if cfg.Format == "console" {
		// Human-readable console output for development
		output := zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		zlog = zerolog.New(output).With().Timestamp().Caller().Logger()
	} else {
		// Structured JSON for production
		zlog = zerolog.New(cfg.Output).With().Timestamp().Caller().Logger()
	}

	return &Logger{zlog: zlog}
}

// WithContext adds logger to context
func (l *Logger) WithContext(ctx context.Context) context.Context {
	return l.zlog.WithContext(ctx)
}

// FromContext retrieves logger from context
func FromContext(ctx context.Context) *Logger {
	zlog := zerolog.Ctx(ctx)
	if zlog.GetLevel() == zerolog.Disabled {
		// Return default logger if not in context
		return New(nil)
	}
	return &Logger{zlog: *zlog}
}

// With creates a child logger with additional fields
func (l *Logger) With() *Context {
	return &Context{ctx: l.zlog.With()}
}

// Context wraps zerolog.Context for field chaining
type Context struct {
	ctx zerolog.Context
}

func (c *Context) Str(key, val string) *Context {
	c.ctx = c.ctx.Str(key, val)
	return c
}

func (c *Context) Int(key string, val int) *Context {
	c.ctx = c.ctx.Int(key, val)
	return c
}

func (c *Context) Err(err error) *Context {
	c.ctx = c.ctx.Err(err)
	return c
}

func (c *Context) Any(key string, val interface{}) *Context {
	c.ctx = c.ctx.Interface(key, val)
	return c
}

func (c *Context) Logger() *Logger {
	return &Logger{zlog: c.ctx.Logger()}
}

// Logging methods
func (l *Logger) Debug(msg string) {
	l.zlog.Debug().Msg(msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zlog.Debug().Msgf(format, args...)
}

func (l *Logger) Info(msg string) {
	l.zlog.Info().Msg(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.zlog.Info().Msgf(format, args...)
}

func (l *Logger) Warn(msg string) {
	l.zlog.Warn().Msg(msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zlog.Warn().Msgf(format, args...)
}

func (l *Logger) Error(msg string) {
	l.zlog.Error().Msg(msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.zlog.Error().Msgf(format, args...)
}

func (l *Logger) Fatal(msg string) {
	l.zlog.Fatal().Msg(msg)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zlog.Fatal().Msgf(format, args...)
}

// Structured logging with fields
func (l *Logger) InfoWith(msg string, fields map[string]interface{}) {
	event := l.zlog.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *Logger) ErrorWith(msg string, err error, fields map[string]interface{}) {
	event := l.zlog.Error().Err(err)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// HTTP middleware helper
func (l *Logger) HTTPEvent() *zerolog.Event {
	return l.zlog.Info()
}

// Helper functions
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func getTimeFormat(format string) string {
	switch format {
	case "unix":
		return zerolog.TimeFormatUnix
	case "unixms":
		return zerolog.TimeFormatUnixMs
	case "unixmicro":
		return zerolog.TimeFormatUnixMicro
	default:
		return time.RFC3339
	}
}

// Global logger instance (for convenience)
var global *Logger

func init() {
	global = New(nil)
}

// Global convenience functions
func Debug(msg string) {
	global.Debug(msg)
}

func Info(msg string) {
	global.Info(msg)
}

func Warn(msg string) {
	global.Warn(msg)
}

func Error(msg string) {
	global.Error(msg)
}

func Fatal(msg string) {
	global.Fatal(msg)
}

func SetGlobal(l *Logger) {
	global = l
}
