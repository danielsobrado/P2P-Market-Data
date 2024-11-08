package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string
	OutputPath string
	MaxSize    int // megabytes
	MaxAge     int // days
	MaxBackups int
	Compress   bool
	Debug      bool
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      "info",
		OutputPath: "logs/app.log",
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 5,
		Compress:   true,
		Debug:      false,
	}
}

// NewLogger creates a new configured logger
func NewLogger(cfg *LogConfig) (*zap.Logger, error) {
	if cfg == nil {
		cfg = DefaultLogConfig()
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	// Configure rotation
	rotator := &lumberjack.Logger{
		Filename:   cfg.OutputPath,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("parsing log level: %w", err)
	}

	// Create core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(rotator),
		level,
	)

	// Add development mode if debug is enabled
	var options []zap.Option
	if cfg.Debug {
		options = append(options, zap.Development())
	}
	options = append(options,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// Create logger
	logger := zap.New(core, options...)

	return logger, nil
}

// LoggerWithContext creates a child logger with context fields
func LoggerWithContext(parent *zap.Logger, fields ...zapcore.Field) *zap.Logger {
	return parent.With(fields...)
}

// LogMetrics represents logging metrics
type LogMetrics struct {
	ErrorCount   int64
	WarnCount    int64
	InfoCount    int64
	DebugCount   int64
	LastLogTime  time.Time
	LastLogLevel zapcore.Level
}

// LogMetricsHook implements zapcore.Core to collect logging metrics
type LogMetricsHook struct {
	zapcore.Core
	metrics *LogMetrics
}

// NewLogMetricsHook creates a new metrics collecting hook
func NewLogMetricsHook(core zapcore.Core) (*LogMetricsHook, *LogMetrics) {
	metrics := &LogMetrics{}
	return &LogMetricsHook{
		Core:    core,
		metrics: metrics,
	}, metrics
}

func (h *LogMetricsHook) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Update metrics
	switch entry.Level {
	case zapcore.ErrorLevel:
		h.metrics.ErrorCount++
	case zapcore.WarnLevel:
		h.metrics.WarnCount++
	case zapcore.InfoLevel:
		h.metrics.InfoCount++
	case zapcore.DebugLevel:
		h.metrics.DebugCount++
	}

	h.metrics.LastLogTime = entry.Time
	h.metrics.LastLogLevel = entry.Level

	return h.Core.Write(entry, fields)
}

// RotateLogs triggers log rotation
func RotateLogs(outputPath string) error {
	rotator := &lumberjack.Logger{
		Filename: outputPath,
	}
	if err := rotator.Rotate(); err != nil {
		return fmt.Errorf("rotating logs: %w", err)
	}
	return nil
}

// LogWriter implements io.Writer for compatibility with other logging systems
type LogWriter struct {
	logger *zap.Logger
	level  zapcore.Level
}

// NewLogWriter creates a new log writer
func NewLogWriter(logger *zap.Logger, level zapcore.Level) *LogWriter {
	return &LogWriter{
		logger: logger,
		level:  level,
	}
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	switch w.level {
	case zapcore.ErrorLevel:
		w.logger.Error(string(p))
	case zapcore.WarnLevel:
		w.logger.Warn(string(p))
	case zapcore.InfoLevel:
		w.logger.Info(string(p))
	case zapcore.DebugLevel:
		w.logger.Debug(string(p))
	}
	return len(p), nil
}
