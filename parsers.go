package dslogger

import "go.uber.org/zap/zapcore"

func parseLogLevel(level string) zapcore.Level {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	return zapLevel
}
