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

var levelNames = map[zapcore.Level]string{
	zapcore.DebugLevel: "DEBUG",
	zapcore.InfoLevel:  "INFO ",
	zapcore.WarnLevel:  "WARN ",
	zapcore.ErrorLevel: "ERROR",
}

var levelColors = map[zapcore.Level]struct {
	color, levelStr string
}{
	zapcore.DebugLevel: {clBlue, "DEBUG"},
	zapcore.InfoLevel:  {clCyan, "INFO "},
	zapcore.WarnLevel:  {clYellow, "WARN "},
	zapcore.ErrorLevel: {clRed, "ERROR"},
}

func fixedWidthLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	levelStr, exists := levelNames[level]
	if !exists {
		levelStr = "UNKNW"
	}
	enc.AppendString(levelStr)
}

func fixedWidthColorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	color, levelStr := clReset, "UNKNW"
	if levelData, exists := levelColors[level]; exists {
		color, levelStr = levelData.color, levelData.levelStr
	}
	enc.AppendString(fmt.Sprintf("%s%s%s", color, levelStr, clReset))
}
