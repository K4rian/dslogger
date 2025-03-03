package dslogger

import "go.uber.org/zap/zapcore"

// parseLogLevel converts a string representation of a log level into a zapcore.Level.
// If the level cannot be parsed, it defaults to zapcore.InfoLevel.
func parseLogLevel(level string) zapcore.Level {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	return zapLevel
}
