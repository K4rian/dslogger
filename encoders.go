package dslogger

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

const (
	clRed    = "\033[31m"
	clYellow = "\033[33m"
	clBlue   = "\033[34m"
	clCyan   = "\033[36m"
	clReset  = "\033[0m"
)

func fixedWidthLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var levelStr string
	switch level {
	case zapcore.DebugLevel:
		levelStr = "DEBUG"
	case zapcore.InfoLevel:
		levelStr = "INFO "
	case zapcore.WarnLevel:
		levelStr = "WARN "
	case zapcore.ErrorLevel:
		levelStr = "ERROR"
	default:
		levelStr = "UNKNW"
	}
	enc.AppendString(levelStr)
}

func fixedWidthColorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color, levelStr string
	switch level {
	case zapcore.DebugLevel:
		color, levelStr = clBlue, "DEBUG"
	case zapcore.InfoLevel:
		color, levelStr = clCyan, "INFO "
	case zapcore.WarnLevel:
		color, levelStr = clYellow, "WARN "
	case zapcore.ErrorLevel:
		color, levelStr = clRed, "ERROR"
	default:
		color, levelStr = clReset, "UNKNW"
	}
	enc.AppendString(fmt.Sprintf("%s%s%s", color, levelStr, clReset))
}
