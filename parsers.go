package dslogger

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// parseLogLevel converts a string representation of a log level into a zapcore.Level.
// Returns an error if the level cannot be parsed, so callers can decide whether to
// fail construction or fall back to a default.
func parseLogLevel(level string) (zapcore.Level, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	return zapLevel, nil
}
