package dslogger

import (
	"io"
	"os"
	"reflect"

	"go.uber.org/zap/zapcore"
)

// LogFormat defines the output format for log files.
type LogFormat string

// LevelFormat holds the fixed-width text representation for a log level,
// as well as an optional ANSI color escape sequence.
type LevelFormat struct {
	LevelStr string
	Color    string
}

// Config holds configuration options for the Logger.
type Config struct {
	LogFile       string
	LogFileFormat LogFormat
	MaxSize       int
	MaxBackups    int
	MaxAge        int
	Compress      bool

	// Deprecated: Level is redundant with the level argument to the constructor functions.
	// If the constructor's level argument is empty, this field is used as a fallback.
	// Will be removed in a future major version.
	Level                 string
	ConsoleConfig         zapcore.EncoderConfig
	FileConfig            zapcore.EncoderConfig
	ConsoleSeparator      string
	FieldSeparator        string
	ServiceNameDecorators [2]string
	LevelFormats          map[zapcore.Level]LevelFormat

	// NoColor disables ANSI color codes on the console encoder.
	// If unset (false), the library auto-detects non-TTY stdout and disables color automatically.
	NoColor bool

	// ForceColor overrides the auto-detection and forces ANSI color codes on the console
	// encoder, even when stdout is not a terminal. Takes precedence over NoColor when both are true.
	ForceColor bool

	// FileMode sets the permission bits used when pre-creating the log file. Lumberjack
	// creates rotated backups with its own default (0600). This field only controls the
	// primary log file. A zero value means no pre-creation
	// (lumberjack creates the file with its own defaults).
	FileMode os.FileMode

	// ConsoleWriter overrides the default console output destination (os.Stdout).
	// Set to os.Stderr, a bytes.Buffer, or any io.Writer. When nil, os.Stdout is used
	ConsoleWriter io.Writer

	// ConsoleFormat controls the console output format. Defaults to LogFormatText
	// (the human-readable dslogger format).
	// Set to LogFormatJSON for structured JSON on stdout
	ConsoleFormat LogFormat
}

// Supported log file formats.
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// cloneConfig returns a deep copy of in. The returned *Config shares no slice or
// map backing with the input, so subsequent mutation of either side is independent.
// A nil input yields a fresh default config.
func cloneConfig(in *Config) *Config {
	if in == nil {
		c := NewDefaultConfig()
		return &c
	}
	c := *in
	if in.LevelFormats != nil {
		c.LevelFormats = make(map[zapcore.Level]LevelFormat, len(in.LevelFormats))
		for k, v := range in.LevelFormats {
			c.LevelFormats[k] = v
		}
	}
	return &c
}

// applyDefaults fills any zero-valued fields on cfg with values from a fresh default config.
// The caller must ensure cfg is already a clone (see cloneConfig), applyDefaults assumes
// it is safe to mutate cfg in place and never writes default-owned maps or slices into it.
func applyDefaults(cfg *Config) {
	d := NewDefaultConfig()

	if cfg.LogFile == "" {
		cfg.LogFile = d.LogFile
	}
	if cfg.Level == "" {
		cfg.Level = d.Level
	}
	if cfg.ConsoleSeparator == "" {
		cfg.ConsoleSeparator = d.ConsoleSeparator
	}
	if cfg.FieldSeparator == "" {
		cfg.FieldSeparator = d.FieldSeparator
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = d.MaxSize
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = d.MaxBackups
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = d.MaxAge
	}
	if cfg.LogFileFormat == "" {
		cfg.LogFileFormat = d.LogFileFormat
	}
	if reflect.DeepEqual(cfg.ConsoleConfig, zapcore.EncoderConfig{}) {
		cfg.ConsoleConfig = d.ConsoleConfig
	}
	if reflect.DeepEqual(cfg.FileConfig, zapcore.EncoderConfig{}) {
		cfg.FileConfig = d.FileConfig
	}
	if cfg.ServiceNameDecorators[0] == "" && cfg.ServiceNameDecorators[1] == "" {
		cfg.ServiceNameDecorators = d.ServiceNameDecorators
	}
	if cfg.LevelFormats == nil {
		cfg.LevelFormats = d.LevelFormats // d is locally owned
	}

	// Auto-detect non-TTY stdout and disable color unless explicitly configured
	// ForceColor takes precedence: if set, ensure NoColor is false regardless of TTY
	if cfg.ForceColor {
		cfg.NoColor = false
	} else if !cfg.NoColor && !stdoutIsTerminal() {
		cfg.NoColor = true
	}

	cfg.ConsoleConfig.ConsoleSeparator = cfg.ConsoleSeparator
	cfg.ConsoleConfig.EncodeLevel = FixedWidthCapitalColorLevelEncoder(cfg)

	// For JSON file output, use the standard level encoder (no fixed-width padding)
	// For text file output, use the fixed-width encoder matching the console format
	cfg.FileConfig.ConsoleSeparator = cfg.ConsoleSeparator
	if cfg.LogFileFormat == LogFormatJSON {
		cfg.FileConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		cfg.FileConfig.EncodeLevel = FixedWidthCapitalLevelEncoder(cfg)
	}
}

// consoleOut returns the console writer, defaulting to os.Stdout.
func (c *Config) consoleOut() io.Writer {
	if c.ConsoleWriter != nil {
		return c.ConsoleWriter
	}
	return os.Stdout
}

// stdoutIsTerminal reports whether os.Stdout is attached to a terminal (character device).
// When stdout is redirected to a file or pipe, ANSI color codes should be suppressed.
func stdoutIsTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
