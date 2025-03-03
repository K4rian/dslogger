package dslogger

import (
	"fmt"
	"os"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// WithCallerSkip adjusts the caller skip level for accurate file/line reporting.
func WithCallerSkip(skip int) Option {
	return func(l *Logger) error {
		if l.consoleLogger != nil {
			l.consoleLogger = l.consoleLogger.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar()
		}
		if l.fileLogger != nil {
			l.fileLogger = l.fileLogger.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar()
		}
		return nil
	}
}

// WithConsoleEncoder sets a custom zapcore.Encoder for the console logger.
func WithConsoleEncoder(encoder zapcore.Encoder) Option {
	return func(l *Logger) error {
		if l.consoleLogger == nil {
			return nil
		}

		logLevel := zapcore.Level(atomic.LoadInt32(&l.level))
		consoleWriteSyncer := zapcore.Lock(os.Stdout) // Console output

		newConsoleCore := zapcore.NewCore(encoder, consoleWriteSyncer, logLevel)
		l.consoleLogger = zap.New(newConsoleCore).Sugar()
		return nil
	}
}

// WithFileEncoder sets a custom zapcore.Encoder for the file logger.
func WithFileEncoder(encoder zapcore.Encoder) Option {
	return func(l *Logger) error {
		if l.fileLogger == nil || l.lumberjackLogger == nil {
			return nil
		}

		logLevel := zapcore.Level(atomic.LoadInt32(&l.level))
		fileWriteSyncer := zapcore.AddSync(l.lumberjackLogger) // File output

		newFileCore := zapcore.NewCore(encoder, fileWriteSyncer, logLevel)
		l.fileLogger = zap.New(newFileCore).Sugar()
		return nil
	}
}

// WithCustomField is an option to attach a custom field to every log.
func WithCustomField(key string, value any) Option {
	return WithCustomFields(key, value)
}

// WithCustomFields attaches multiple custom fields to every log entry.
func WithCustomFields(fields ...any) Option {
	return func(l *Logger) error {
		if len(fields)%2 != 0 {
			return fmt.Errorf("WithCustomFields requires an even number of arguments (key-value pairs)")
		}

		zapFields := []zap.Field{}
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				return fmt.Errorf("field key must be a string, got %T", fields[i])
			}
			value := fields[i+1]
			zapFields = append(zapFields, zap.Any(key, value))
		}

		// Store fields in the logger struct
		l.customFields = append(l.customFields, zapFields...)

		// Apply fields to file logger
		if l.fileLogger != nil {
			l.fileLogger = l.fileLogger.With(zapFieldsToAny(zapFields)...)
		}
		return nil
	}
}

// WithCustomLevelFormats is an option that sets custom level formats on the logger's config.
func WithCustomLevelFormats(formats map[zapcore.Level]LevelFormat) Option {
	return func(l *Logger) error {
		// Merge with existing level formats (or overwrite as desired)
		if l.config.LevelFormats == nil {
			l.config.LevelFormats = formats
		} else {
			for level, lf := range formats {
				l.config.LevelFormats[level] = lf
			}
		}
		return nil
	}
}

// WithServiceName is an option to set the service name.
func WithServiceName(name string) Option {
	return func(l *Logger) error {
		l.serviceName = name
		return nil
	}
}
