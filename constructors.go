package dslogger

import (
	"fmt"
	"os"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewConsoleLogger creates a new logger that outputs only to the console.
// It applies any provided functional options for additional configuration.
func NewConsoleLogger(level string, config *Config, opts ...Option) (*Logger, error) {
	return newLogger(level, config, false, opts...)
}

// NewLogger creates a new logger that outputs to both the console and a file.
// It uses the provided configuration and applies any functional options.
func NewLogger(level string, config *Config, opts ...Option) (*Logger, error) {
	return newLogger(level, config, true, opts...)
}

// NewSimpleConsoleLogger creates a logger that outputs only to the console using default configuration.
func NewSimpleConsoleLogger(level string) (*Logger, error) {
	return newLogger(level, &DefaultLoggerConfig, false)
}

// NewSimpleLogger creates a logger that outputs to both console and a text file using default configuration.
func NewSimpleLogger(level string) (*Logger, error) {
	return newLogger(level, &DefaultLoggerConfig, true)
}

// newConsoleLogger is an internal helper that creates a console logger given the configuration and level.
func newConsoleLogger(config *Config, level zapcore.Level) (*zap.SugaredLogger, error) {
	encoder := zapcore.NewConsoleEncoder(config.ConsoleConfig)
	return newZapLogger(encoder, zapcore.AddSync(os.Stdout), level)
}

// newFileLogger is an internal helper that creates a file logger using the provided lumberjack logger.
func newFileLogger(config *Config, level zapcore.Level, ljLogger *lumberjack.Logger) (*zap.SugaredLogger, error) {
	var encoder zapcore.Encoder

	if config.LogFileFormat == LogFormatText {
		encoder = zapcore.NewConsoleEncoder(config.FileConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(config.FileConfig)
	}
	return newZapLogger(encoder, zapcore.AddSync(ljLogger), level)
}

// newZapLogger is a helper function that creates a zap logger given an encoder, write syncer, and log level.
func newZapLogger(encoder zapcore.Encoder, syncer zapcore.WriteSyncer, level zapcore.Level) (*zap.SugaredLogger, error) {
	core := zapcore.NewCore(encoder, syncer, level)
	logger := zap.New(core, zap.AddCaller()).Sugar()

	if logger == nil {
		return nil, fmt.Errorf("failed to create zap logger")
	}
	return logger, nil
}

// newLogger is a helper function that creates a new logger instance.
// If fileLogging is true, it sets up a file logger using lumberjack for file rotation;
// otherwise, it only creates a console logger. It also applies any provided functional options.
func newLogger(level string, config *Config, fileLogging bool, opts ...Option) (*Logger, error) {
	var err error

	// Merge user configuration with defaults
	if config == nil {
		config = &DefaultLoggerConfig
	} else {
		*config = *mergeConfig(config)
	}

	zapLevel := parseLogLevel(level)
	consoleLogger, err := newConsoleLogger(config, zapLevel)
	if err != nil {
		return nil, err
	}

	var fileLogger *zap.SugaredLogger
	var ljLogger *lumberjack.Logger

	if fileLogging {
		// Create a lumberjack logger for file rotation
		ljLogger = &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		fileLogger, err = newFileLogger(config, zapLevel, ljLogger)
		if err != nil {
			return nil, err
		}
	}

	logger := &Logger{
		config:           config,
		consoleLogger:    consoleLogger,
		fileLogger:       fileLogger,
		lumberjackLogger: ljLogger,
		serviceName:      "",
	}
	atomic.StoreInt32(&logger.level, int32(zapLevel))

	initializeLogFuncMaps(logger)

	// Apply functional options
	for _, opt := range opts {
		if err := opt(logger); err != nil {
			return nil, err
		}
	}
	return logger, nil
}
