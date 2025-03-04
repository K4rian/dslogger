package dslogger

import (
	"reflect"

	"go.uber.org/zap/zapcore"
)

// LogFormat defines the output format for log files.
type LogFormat string

// LevelFormat holds the fixed-width text representation for a log level,
// as well as an optional ANSI color escape sequence.
type LevelFormat struct {
	LevelStr string // e.g. "DEBUG", "INFO", etc.
	Color    string // e.g. "\033[34m" for blue; empty string if no color.
}

// Config holds configuration options for the Logger.
type Config struct {
	LogFile               string
	LogFileFormat         LogFormat
	MaxSize               int
	MaxBackups            int
	MaxAge                int
	Compress              bool
	Level                 string
	ConsoleConfig         zapcore.EncoderConfig
	FileConfig            zapcore.EncoderConfig
	ConsoleSeparator      string
	FieldSeparator        string
	ServiceNameDecorators [2]string
	LevelFormats          map[zapcore.Level]LevelFormat
}

// Supported log file formats.
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// mergeConfig manually combines a user-provided configuration with the defaults.
// For each field, if the user provided a zero value, the default is used.
func mergeConfig(userConfig *Config) *Config {
	if userConfig.LogFile == "" {
		userConfig.LogFile = DefaultLoggerConfig.LogFile
	}
	if userConfig.Level == "" {
		userConfig.Level = DefaultLoggerConfig.Level
	}
	if userConfig.ConsoleSeparator == "" {
		userConfig.ConsoleSeparator = DefaultLoggerConfig.ConsoleSeparator
	}
	if userConfig.FieldSeparator == "" {
		userConfig.FieldSeparator = DefaultLoggerConfig.FieldSeparator
	}

	if userConfig.MaxSize == 0 {
		userConfig.MaxSize = DefaultLoggerConfig.MaxSize
	}
	if userConfig.MaxBackups == 0 {
		userConfig.MaxBackups = DefaultLoggerConfig.MaxBackups
	}
	if userConfig.MaxAge == 0 {
		userConfig.MaxAge = DefaultLoggerConfig.MaxAge
	}

	if userConfig.LogFileFormat == "" {
		userConfig.LogFileFormat = DefaultLoggerConfig.LogFileFormat
	}

	if reflect.DeepEqual(userConfig.ConsoleConfig, zapcore.EncoderConfig{}) {
		userConfig.ConsoleConfig = DefaultLoggerConfig.ConsoleConfig
	}

	if reflect.DeepEqual(userConfig.FileConfig, zapcore.EncoderConfig{}) {
		userConfig.FileConfig = DefaultLoggerConfig.FileConfig
	}

	if userConfig.ServiceNameDecorators[0] == "" && userConfig.ServiceNameDecorators[1] == "" {
		userConfig.ServiceNameDecorators = DefaultLoggerConfig.ServiceNameDecorators
	}

	if userConfig.LevelFormats == nil {
		userConfig.LevelFormats = DefaultLoggerConfig.LevelFormats
	}

	if userConfig.ConsoleConfig.ConsoleSeparator != userConfig.ConsoleSeparator {
		userConfig.ConsoleConfig.ConsoleSeparator = userConfig.ConsoleSeparator
	}
	userConfig.ConsoleConfig.EncodeLevel = FixedWidthCapitalColorLevelEncoder(userConfig)

	if userConfig.FileConfig.ConsoleSeparator != userConfig.ConsoleSeparator {
		userConfig.FileConfig.ConsoleSeparator = userConfig.ConsoleSeparator
	}
	userConfig.FileConfig.EncodeLevel = FixedWidthCapitalLevelEncoder(userConfig)

	return userConfig
}
