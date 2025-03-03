package dslogger

import "go.uber.org/zap/zapcore"

var (
	DefaultLevelFormats = map[zapcore.Level]LevelFormat{
		zapcore.DebugLevel: {LevelStr: "DEBUG", Color: "\033[34m"},
		zapcore.InfoLevel:  {LevelStr: "INFO ", Color: "\033[36m"},
		zapcore.WarnLevel:  {LevelStr: "WARN ", Color: "\033[33m"},
		zapcore.ErrorLevel: {LevelStr: "ERROR", Color: "\033[31m"},
	}

	// DefaultConsoleEncoderConfig defines the default encoder configuration for console output.
	// It uses ISO8601 for time encoding, a fixed-width colored encoder for log levels,
	// and a short caller encoder to show the file and line number.
	DefaultConsoleEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// DefaultTextEncoderConfig defines the default encoder configuration for plain text log output.
	// It uses ISO8601 for time encoding, a fixed-width log level encoder (without color),
	// and a short caller encoder.
	DefaultTextEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// DefaultJSONEncoderConfig defines the default encoder configuration for JSON log output.
	// It uses ISO8601 for time encoding, a capitalized log level encoder,
	// and a short caller encoder to include caller information.
	DefaultJSONEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// DefaultLoggerConfig provides the default configuration for a dslogger.Logger instance.
	// It sets default values for the log file path, file format, rotation policies,
	// encoder configurations for both console and file outputs, and other display options.
	DefaultLoggerConfig = Config{
		LogFile:               "app.log",                   // Default log file path
		LogFileFormat:         LogFormatText,               // Default log file format (text)
		MaxSize:               10,                          // Maximum size (in MB) of the log file before rotation
		MaxBackups:            5,                           // Maximum number of backup log files to retain
		MaxAge:                28,                          // Maximum age (in days) to retain a backup log file
		Compress:              true,                        // Whether to compress rotated log files
		Level:                 "info",                      // Default log level
		ConsoleConfig:         DefaultConsoleEncoderConfig, // Encoder configuration for console logs
		FileConfig:            DefaultTextEncoderConfig,    // Encoder configuration for file logs
		FieldSeparator:        ": ",                        // Separator between key and value in log fields
		ConsoleSeparator:      " | ",                       // Separator between fields in console output
		ServiceNameDecorators: [2]string{"[", "]"},         // Decorators to wrap the service name in log messages
		LevelFormats:          DefaultLevelFormats,         // Default Level formats
	}
)
