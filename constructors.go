package dslogger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewConsoleLogger creates a new logger that outputs only to the console.
func NewConsoleLogger(level string, config *Config, opts ...Option) (*Logger, error) {
	return newLogger(level, config, false, opts...)
}

// NewLogger creates a new logger that outputs to both the console and a file.
func NewLogger(level string, config *Config, opts ...Option) (*Logger, error) {
	return newLogger(level, config, true, opts...)
}

// NewSimpleConsoleLogger creates a console-only logger using default configuration.
func NewSimpleConsoleLogger(level string) (*Logger, error) {
	return newLogger(level, nil, false)
}

// NewSimpleLogger creates a logger writing to console and a text file using default configuration.
func NewSimpleLogger(level string) (*Logger, error) {
	return newLogger(level, nil, true)
}

// buildConsoleZap creates a zap SugaredLogger for console output.
// When ConsoleFormat is LogFormatJSON it uses zap's stock JSON encoder,
// otherwise it uses the custom dsConsoleEncoder.
// The writer is wrapped in zapcore.Lock so that concurrent writers
// cannot produce torn/garbage output.
func buildConsoleZap(cfg *Config, level zap.AtomicLevel, serviceName string) *zap.SugaredLogger {
	var encoder zapcore.Encoder

	if cfg.ConsoleFormat == LogFormatJSON {
		encoder = zapcore.NewJSONEncoder(cfg.ConsoleConfig)
	} else {
		encoder = newDSConsoleEncoder(cfg, cfg.ConsoleConfig, serviceName)
	}
	writer := zapcore.Lock(zapcore.AddSync(cfg.consoleOut()))
	core := zapcore.NewCore(encoder, writer, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)).Sugar()
}

// buildFileZap creates a zap SugaredLogger wrapping the given lumberjack.Logger.
// For JSON format it uses zap's stock JSON encoder,
// for text format it uses the custom dsConsoleEncoder (same visual as console, minus colour).
func buildFileZap(cfg *Config, level zap.AtomicLevel, ljLogger *lumberjack.Logger, serviceName string) *zap.SugaredLogger {
	var encoder zapcore.Encoder

	if cfg.LogFileFormat == LogFormatJSON {
		encoder = zapcore.NewJSONEncoder(cfg.FileConfig)
	} else {
		encoder = newDSConsoleEncoder(cfg, cfg.FileConfig, serviceName)
	}
	core := zapcore.NewCore(encoder, zapcore.AddSync(ljLogger), level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)).Sugar()
}

// newLogger is the shared constructor, it always deep-copies the caller's config,
// applies defaults to the copy, and never mutates user-owned state.
func newLogger(level string, config *Config, fileLogging bool, opts ...Option) (*Logger, error) {
	cfg := cloneConfig(config)
	applyDefaults(cfg)

	// Fall back to Config.Level when the explicit argument is empty
	if level == "" && cfg.Level != "" {
		level = cfg.Level
	}

	parsedLevel, err := parseLogLevel(level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dslogger: %v; falling back to info level\n", err)
		parsedLevel = zapcore.InfoLevel
	}
	atomicLvl := zap.NewAtomicLevelAt(parsedLevel)

	logger := &Logger{
		config:      cfg,
		level:       atomicLvl,
		serviceName: "",
	}
	logger.consoleLogger.Store(buildConsoleZap(cfg, atomicLvl, ""))

	if fileLogging {
		// Pre-create the log file with the configured permissions so that
		// lumberjack (which defaults to 0600 internally) inherits our mode
		if cfg.FileMode != 0 {
			if err := ensureFileMode(cfg.LogFile, cfg.FileMode); err != nil {
				return nil, fmt.Errorf("dslogger: pre-create log file: %w", err)
			}
		}

		logger.lumberjackLogger = &lumberjack.Logger{
			Filename:   cfg.LogFile,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		logger.fileLogger.Store(buildFileZap(cfg, atomicLvl, logger.lumberjackLogger, ""))
	}

	for _, opt := range opts {
		if err := opt(logger); err != nil {
			return nil, err
		}
	}
	return logger, nil
}

// ensureFileMode creates or chmods the given file to the requested permissions.
func ensureFileMode(path string, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, mode)
	if err != nil {
		return err
	}
	return f.Close()
}
