package dslogger

import "go.uber.org/zap/zapcore"

// defaultLevelFormats returns a fresh map of default level formats on every call.
// Returning a new map ensures that callers and loggers can't accidentally share
// or mutate package-level state.
func defaultLevelFormats() map[zapcore.Level]LevelFormat {
	return map[zapcore.Level]LevelFormat{
		zapcore.DebugLevel: {LevelStr: "DEBUG", Color: "\033[34m"},
		zapcore.InfoLevel:  {LevelStr: "INFO ", Color: "\033[36m"},
		zapcore.WarnLevel:  {LevelStr: "WARN ", Color: "\033[33m"},
		zapcore.ErrorLevel: {LevelStr: "ERROR", Color: "\033[31m"},
	}
}

var (
	// DefaultLevelFormats is kept for backwards compatibility.
	// Prefer defaultLevelFormats() or NewDefaultConfig().
	// Direct mutation of this map is unsafe and is not reflected in newly constructed loggers.
	DefaultLevelFormats = defaultLevelFormats()

	// DefaultConsoleEncoderConfig defines the default encoder configuration for console output.
	// The EncodeLevel field is intentionally left unset here, applyDefaults sets the fixed-width
	// color level encoder bound to the target logger's Config.
	DefaultConsoleEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// DefaultTextEncoderConfig defines the default encoder configuration for plain text log output.
	DefaultTextEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// DefaultJSONEncoderConfig defines the default encoder configuration for JSON log output.
	DefaultJSONEncoderConfig = zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
)

// NewDefaultConfig returns a fresh Config populated with sensible defaults.
// Every call returns a fully independent value, mutating the result is safe
// and has no effect on other loggers.
func NewDefaultConfig() Config {
	return Config{
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
		LevelFormats:          defaultLevelFormats(),       // Default Level formats
		FileMode:              0600,                        // Log file mode
	}
}

// DefaultLoggerConfig is kept for backwards compatibility. Prefer NewDefaultConfig().
// The value is deep-copied during logger construction, so mutating this variable
// does not affect already-running loggers, but it is still shared package state,
// so concurrent mutation is unsafe.
var DefaultLoggerConfig = NewDefaultConfig()
