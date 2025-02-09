package dslogger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	config           *Config
	consoleLogger    *zap.SugaredLogger
	fileLogger       *zap.SugaredLogger
	lumberjackLogger *lumberjack.Logger
	serviceName      string
}

func (l *Logger) Config() *Config {
	return l.config
}

func (l *Logger) ConsoleLogger() *zap.SugaredLogger {
	return l.consoleLogger
}

func (l *Logger) FileLogger() *zap.SugaredLogger {
	return l.fileLogger
}

func (l *Logger) LumberjackLogger() *lumberjack.Logger {
	return l.lumberjackLogger
}

func (l *Logger) ServiceName() string {
	return l.serviceName
}

func (l *Logger) SetLogLevel(level string) {
	zapLevel := parseLogLevel(level)

	consoleEncoder := zapcore.NewConsoleEncoder(l.config.ConsoleConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel)

	l.consoleLogger = zap.New(consoleCore, zap.AddCaller()).Sugar()

	// Recreate file logger if enabled
	if l.fileLogger != nil {
		var fileEncoder zapcore.Encoder
		if l.config.LogFileFormat == LogFormatText {
			fileEncoder = zapcore.NewConsoleEncoder(l.config.FileConfig)
		} else {
			fileEncoder = zapcore.NewJSONEncoder(l.config.FileConfig)
		}
		fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(l.lumberjackLogger), zapLevel)
		l.fileLogger = zap.New(fileCore, zap.AddCaller()).Sugar()
	}
}

func (l *Logger) WithService(serviceName string, options ...zap.Option) *Logger {
	newConsoleLogger := l.consoleLogger.Desugar().WithOptions(options...).Sugar()
	newFileLogger := l.fileLogger

	// Only attach "service" to the file logger if it's in JSON format
	if l.fileLogger != nil && l.config.LogFileFormat == LogFormatJSON {
		newFileLogger = l.fileLogger.Desugar().With(zap.String("service", serviceName)).WithOptions(options...).Sugar()
	}

	return &Logger{
		config:           l.config,
		consoleLogger:    newConsoleLogger,
		fileLogger:       newFileLogger,
		lumberjackLogger: l.lumberjackLogger,
		serviceName:      serviceName,
	}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.logMessage("info", msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.logMessage("debug", msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.logMessage("warn", msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.logMessage("error", msg, fields...)
}

func (l *Logger) logMessage(level string, msg string, fields ...interface{}) {
	formattedMsg := l.formatMessage(msg, fields...)

	// Console log functions
	consoleLoggers := map[string]func(...interface{}){
		"info":  l.consoleLogger.Info,
		"debug": l.consoleLogger.Debug,
		"warn":  l.consoleLogger.Warn,
		"error": l.consoleLogger.Error,
	}

	// Log to console
	if logFunc, exists := consoleLoggers[level]; exists {
		logFunc(formattedMsg)
	}

	// Log to file if enabled
	if l.fileLogger != nil {
		if l.config.LogFileFormat == LogFormatText {
			// File log functions (text format)
			fileLoggers := map[string]func(...interface{}){
				"info":  l.fileLogger.Info,
				"debug": l.fileLogger.Debug,
				"warn":  l.fileLogger.Warn,
				"error": l.fileLogger.Error,
			}
			if logFunc, exists := fileLoggers[level]; exists {
				logFunc(formattedMsg)
			}
		} else {
			// File log functions (JSON format)
			fileStructuredLoggers := map[string]func(string, ...interface{}){
				"info":  l.fileLogger.Infow,
				"debug": l.fileLogger.Debugw,
				"warn":  l.fileLogger.Warnw,
				"error": l.fileLogger.Errorw,
			}
			if logFunc, exists := fileStructuredLoggers[level]; exists {
				logFunc(msg, fields...)
			}
		}
	}
}
