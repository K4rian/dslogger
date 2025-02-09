package dslogger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewConsoleLogger creates a logger that outputs only to the console.
func NewConsoleLogger(level string, config *Config) *Logger {
	if config != &DefaultLoggerConfig {
		config = mergeConfig(config)
	}
	config.ConsoleConfig.ConsoleSeparator = config.ConsoleSeparator

	zapLevel := parseLogLevel(level)

	consoleEncoder := zapcore.NewConsoleEncoder(config.ConsoleConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel)

	return &Logger{
		config:           config,
		consoleLogger:    zap.New(consoleCore, zap.AddCaller()).Sugar(),
		fileLogger:       nil,
		lumberjackLogger: nil,
		serviceName:      "",
	}
}

// NewLogger creates a logger that outputs to both console and a file (text or JSON).
func NewLogger(level string, config *Config) *Logger {
	if config != &DefaultLoggerConfig {
		config = mergeConfig(config)
	}
	config.ConsoleConfig.ConsoleSeparator = config.ConsoleSeparator

	zapLevel := parseLogLevel(level)

	consoleEncoder := zapcore.NewConsoleEncoder(config.ConsoleConfig)

	var fileEncoder zapcore.Encoder
	if config.LogFileFormat == LogFormatJSON {
		fileEncoder = zapcore.NewJSONEncoder(config.FileConfig)
	} else {
		config.FileConfig.ConsoleSeparator = config.ConsoleSeparator
		fileEncoder = zapcore.NewConsoleEncoder(config.FileConfig)
	}

	ljLogger := &lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(ljLogger), zapLevel)

	consoleLogger := zap.New(consoleCore, zap.AddCaller()).Sugar()
	fileLogger := zap.New(fileCore, zap.AddCaller()).Sugar()

	return &Logger{
		config:           config,
		consoleLogger:    consoleLogger,
		fileLogger:       fileLogger,
		lumberjackLogger: ljLogger,
		serviceName:      "",
	}
}

// NewSimpleConsoleLogger creates a logger that outputs only to the console using default configuration.
func NewSimpleConsoleLogger(level string) *Logger {
	return NewConsoleLogger(level, &DefaultLoggerConfig)
}

// NewSimpleLogger creates a logger that outputs to both console and a text file using default configuration.
func NewSimpleLogger(level string) *Logger {
	return NewLogger(level, &DefaultLoggerConfig)
}
