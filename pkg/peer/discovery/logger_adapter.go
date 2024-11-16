// pkg/peer/discovery/logger_adapter.go
package discovery

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger defines common logging interface
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	With(fields ...interface{}) Logger
}

// ZapLoggerAdapter adapts zap.Logger to Logger interface
type ZapLoggerAdapter struct {
	log *zap.Logger
}

func NewZapLoggerAdapter(log *zap.Logger) *ZapLoggerAdapter {
	return &ZapLoggerAdapter{log: log}
}

func (l *ZapLoggerAdapter) Debug(msg string, fields ...interface{}) {
	l.log.Debug(msg, toZapFields(fields...)...)
}

func (l *ZapLoggerAdapter) Info(msg string, fields ...interface{}) {
	l.log.Info(msg, toZapFields(fields...)...)
}

func (l *ZapLoggerAdapter) Warn(msg string, fields ...interface{}) {
	l.log.Warn(msg, toZapFields(fields...)...)
}

func (l *ZapLoggerAdapter) Error(msg string, fields ...interface{}) {
	l.log.Error(msg, toZapFields(fields...)...)
}

func (l *ZapLoggerAdapter) With(fields ...interface{}) Logger {
	return &ZapLoggerAdapter{log: l.log.With(toZapFields(fields...)...)}
}

// Helper to convert interface fields to zap.Field
func toZapFields(fields ...interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			zapFields = append(zapFields, zap.Any(fmt.Sprint(fields[i]), fields[i+1]))
		}
	}
	return zapFields
}
