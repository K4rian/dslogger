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

// mergeConfig combines user-provided configuration with the default configuration.
// Any field in userConfig that is considered empty will be replaced with the default value.
func mergeConfig(userConfig *Config) *Config {
	userValues := reflect.ValueOf(userConfig).Elem()
	defaultValues := reflect.ValueOf(DefaultLoggerConfig)

	for i := 0; i < userValues.NumField(); i++ {
		field := userValues.Field(i)
		if isEmptyValue(field) {
			field.Set(defaultValues.Field(i))
		}
	}
	return userConfig
}

// isEmptyValue checks whether a reflected value is considered empty.
// It supports common types such as strings, ints, booleans, floats, structs, slices, and arrays.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	case reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isEmptyValue(v.Index(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
