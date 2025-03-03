package dslogger

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// fixedWidthLevelEncoder encodes the log level as a fixed-width string.
// This is used to ensure consistent alignment in console output.
func fixedWidthLevelEncoder(cfg *Config) zapcore.LevelEncoder {
	return levelEncoder(cfg, false)
}

// fixedWidthColorLevelEncoder encodes the log level as a colored, fixed-width string.
// It uses ANSI escape codes to colorize the output.
func fixedWidthColorLevelEncoder(cfg *Config) zapcore.LevelEncoder {
	return levelEncoder(cfg, true)
}

// levelEncoder returns a zapcore.LevelEncoder that uses the provided config's LevelFormats.
// If a color is defined for the level, the output will include ANSI color codes.
func levelEncoder(cfg *Config, colored bool) zapcore.LevelEncoder {
	return func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		lf, ok := cfg.LevelFormats[level]
		if !ok {
			// Fallback if no custom format is provided.
			lf = LevelFormat{LevelStr: level.String()}
		}
		// If a color is provided, wrap the level string with the color codes.
		if colored {
			enc.AppendString(fmt.Sprintf("%s%s%s", lf.Color, lf.LevelStr, "\033[0m"))
		} else {
			enc.AppendString(lf.LevelStr)
		}
	}
}
