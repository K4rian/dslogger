package dslogger

import (
	"fmt"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// WithCallerSkip adjusts the caller skip level for accurate file/line reporting.
// Added on top of the library's baseline skip (dsloggerCallerSkip).
func WithCallerSkip(skip int) Option {
	return func(l *Logger) error {
		if c := l.consoleLogger.Load(); c != nil {
			l.consoleLogger.Store(c.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar())
		}
		if f := l.fileLogger.Load(); f != nil {
			l.fileLogger.Store(f.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar())
		}
		return nil
	}
}

// WithConsoleEncoder sets a custom zapcore.Encoder for the console logger.
// The stdout writer remains wrapped in zapcore.Lock.
func WithConsoleEncoder(encoder zapcore.Encoder) Option {
	return func(l *Logger) error {
		if l.consoleLogger.Load() == nil {
			return nil
		}
		writer := zapcore.Lock(zapcore.AddSync(l.config.consoleOut()))
		core := zapcore.NewCore(encoder, writer, l.level)
		l.consoleLogger.Store(zap.New(core, zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)).Sugar())
		return nil
	}
}

// WithFileEncoder sets a custom zapcore.Encoder for the file logger.
func WithFileEncoder(encoder zapcore.Encoder) Option {
	return func(l *Logger) error {
		if l.fileLogger.Load() == nil || l.lumberjackLogger == nil {
			return nil
		}
		core := zapcore.NewCore(encoder, zapcore.AddSync(l.lumberjackLogger), l.level)
		l.fileLogger.Store(zap.New(core, zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)).Sugar())
		return nil
	}
}

// WithCustomField is an option to attach a single custom field to every log.
func WithCustomField(key string, value any) Option {
	return WithCustomFields(key, value)
}

// WithCustomFields attaches multiple custom fields to every log entry.
// Fields are applied to both loggers via zap's With(), so the custom encoder
// (console/text) and JSON encoder both receive them structurally.
func WithCustomFields(fields ...any) Option {
	return func(l *Logger) error {
		if len(fields)%2 != 0 {
			return fmt.Errorf("WithCustomFields requires an even number of arguments (key-value pairs)")
		}
		zapFields := make([]zap.Field, 0, len(fields)/2)
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				return fmt.Errorf("field key must be a string, got %T", fields[i])
			}
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}

		// Create a fresh slice to avoid aliasing with any derived loggers.
		l.customFields = append(slices.Clone(l.customFields), zapFields...)

		// Apply to both loggers via zap's With so the encoder receives them.
		if c := l.consoleLogger.Load(); c != nil {
			l.consoleLogger.Store(c.Desugar().With(zapFields...).Sugar())
		}
		if f := l.fileLogger.Load(); f != nil {
			l.fileLogger.Store(f.Desugar().With(zapFields...).Sugar())
		}
		return nil
	}
}

// WithCustomLevelFormats sets custom level formats. Because the level encoder
// snapshots its LevelFormats at construction time, this option rebuilds the
// console and file zap cores so that the new formats take effect.
func WithCustomLevelFormats(formats map[zapcore.Level]LevelFormat) Option {
	return func(l *Logger) error {
		l.mu.Lock()
		defer l.mu.Unlock()

		if l.config.LevelFormats == nil {
			l.config.LevelFormats = make(map[zapcore.Level]LevelFormat, len(formats))
		}
		for level, lf := range formats {
			l.config.LevelFormats[level] = lf
		}

		// Rebind the encoders so they snapshot the updated map
		l.config.ConsoleConfig.EncodeLevel = FixedWidthCapitalColorLevelEncoder(l.config)
		l.config.FileConfig.EncodeLevel = FixedWidthCapitalLevelEncoder(l.config)

		// Rebuild both cores with fresh encoders that snapshot the new level formats
		l.consoleLogger.Store(buildConsoleZap(l.config, l.level, l.serviceName))
		if l.lumberjackLogger != nil {
			l.fileLogger.Store(buildFileZap(l.config, l.level, l.lumberjackLogger, l.serviceName))
		}

		// Re-apply custom fields to the rebuilt loggers
		if len(l.customFields) > 0 {
			if c := l.consoleLogger.Load(); c != nil {
				l.consoleLogger.Store(c.Desugar().With(l.customFields...).Sugar())
			}
			if f := l.fileLogger.Load(); f != nil {
				l.fileLogger.Store(f.Desugar().With(l.customFields...).Sugar())
			}
		}
		return nil
	}
}

// WithServiceName is an option to set the service name at construction time.
func WithServiceName(name string) Option {
	return func(l *Logger) error {
		l.serviceName = name
		// Rebuild console and text-file encoders with the service name
		l.consoleLogger.Store(buildConsoleZap(l.config, l.level, name))

		if l.lumberjackLogger != nil && l.config.LogFileFormat != LogFormatJSON {
			l.fileLogger.Store(buildFileZap(l.config, l.level, l.lumberjackLogger, name))
		}

		// For JSON file output, add as a structured field
		if f := l.fileLogger.Load(); f != nil && l.config.LogFileFormat == LogFormatJSON {
			l.fileLogger.Store(f.Desugar().With(zap.String("service", name)).Sugar())
		}

		// Re-apply custom fields
		if len(l.customFields) > 0 {
			if c := l.consoleLogger.Load(); c != nil {
				l.consoleLogger.Store(c.Desugar().With(l.customFields...).Sugar())
			}
			if f := l.fileLogger.Load(); f != nil && l.config.LogFileFormat != LogFormatJSON {
				l.fileLogger.Store(f.Desugar().With(l.customFields...).Sugar())
			}
		}
		return nil
	}
}
