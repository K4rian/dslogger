package dslogger

import (
	"go.uber.org/zap/zapcore"
)

// FixedWidthCapitalLevelEncoder encodes the log level as a fixed-width string.
// The encoder snapshots cfg.LevelFormats at construction time, so the returned
// function is race-free on the hot path and unaffected by subsequent mutation
// of cfg.LevelFormats.
func FixedWidthCapitalLevelEncoder(cfg *Config) zapcore.LevelEncoder {
	return buildLevelEncoder(cfg, false)
}

// FixedWidthCapitalColorLevelEncoder encodes the log level as a colored, fixed-width string.
// Colors are disabled automatically when cfg.NoColor is true which applyDefaults sets when
// stdout is not a terminal.
func FixedWidthCapitalColorLevelEncoder(cfg *Config) zapcore.LevelEncoder {
	return buildLevelEncoder(cfg, true)
}

// precomputedLevel caches the fully-formatted level strings (with and without color)
// so the hot path does not allocate via fmt.Sprintf.
type precomputedLevel struct {
	plain   string
	colored string
}

func buildLevelEncoder(cfg *Config, colored bool) zapcore.LevelEncoder {
	effectiveColored := colored && !cfg.NoColor

	snap := make(map[zapcore.Level]precomputedLevel, len(cfg.LevelFormats))
	for lvl, lf := range cfg.LevelFormats {
		entry := precomputedLevel{plain: lf.LevelStr}
		if lf.Color != "" {
			entry.colored = lf.Color + lf.LevelStr + "\033[0m"
		} else {
			entry.colored = lf.LevelStr
		}
		snap[lvl] = entry
	}

	return func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		p, ok := snap[level]
		if !ok {
			enc.AppendString(level.CapitalString())
			return
		}
		if effectiveColored {
			enc.AppendString(p.colored)
		} else {
			enc.AppendString(p.plain)
		}
	}
}
