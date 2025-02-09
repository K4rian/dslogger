package dslogger

import "go.uber.org/zap/zapcore"

var (
	DefaultConsoleEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  fixedWidthColorLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	DefaultJSONEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	DefaultTextEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  fixedWidthLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	DefaultLoggerConfig = Config{
		LogFile:               "./app.log",
		LogFileFormat:         LogFormatText,
		MaxSize:               10,
		MaxBackups:            5,
		MaxAge:                28,
		Compress:              true,
		Level:                 "info",
		ConsoleConfig:         DefaultConsoleEncoderConfig,
		FileConfig:            DefaultTextEncoderConfig,
		FieldSeparator:        ": ",
		ConsoleSeparator:      " | ",
		ServiceNameDecorators: [2]string{"[", "]"},
	}
)
