package dslogger

import (
	"context"
	"log/slog"
	"slices"

	"go.uber.org/zap/zapcore"
)

// SlogHandler implements [slog.Handler] by delegating to a dslogger [Logger].
// This allows code written against the stdlib slog API to route through
// dslogger's console formatting, file rotation, and level management.
//
// Usage:
//
//	logger, _ := dslogger.NewSimpleConsoleLogger("info")
//	slogger := slog.New(dslogger.NewSlogHandler(logger))
//	slogger.Info("message", "key", "value")
type SlogHandler struct {
	logger *Logger
	attrs  []slog.Attr
	group  string
}

// NewSlogHandler returns a [slog.Handler] backed by the given dslogger Logger.
func NewSlogHandler(logger *Logger) *SlogHandler {
	return &SlogHandler{logger: logger}
}

// Enabled reports whether the handler handles records at the given level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.level.Enabled(slogToZapLevel(level))
}

// Handle formats the record and writes it via the underlying dslogger Logger.
func (h *SlogHandler) Handle(_ context.Context, record slog.Record) error {
	lvl := slogToZapLevel(record.Level)

	// Estimate total field count: handler attrs + record attrs
	n := len(h.attrs)*2 + record.NumAttrs()*2
	fields := make([]any, 0, n)

	for _, a := range h.attrs {
		fields = appendAttr(fields, h.group, a)
	}
	record.Attrs(func(a slog.Attr) bool {
		fields = appendAttr(fields, h.group, a)
		return true
	})

	h.logger.logMessage(lvl, record.Message, fields...)
	return nil
}

// WithAttrs returns a new handler whose output includes the given attributes.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogHandler{
		logger: h.logger,
		attrs:  append(slices.Clone(h.attrs), attrs...),
		group:  h.group,
	}
}

// WithGroup returns a new handler that prefixes subsequent attribute keys with the given group name.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroup := name
	if h.group != "" {
		newGroup = h.group + "." + name
	}
	return &SlogHandler{
		logger: h.logger,
		attrs:  slices.Clone(h.attrs),
		group:  newGroup,
	}
}

// slogToZapLevel maps a slog.Level to the closest zapcore.Level.
func slogToZapLevel(l slog.Level) zapcore.Level {
	switch {
	case l >= slog.LevelError:
		return zapcore.ErrorLevel
	case l >= slog.LevelWarn:
		return zapcore.WarnLevel
	case l >= slog.LevelInfo:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

// appendAttr flattens a slog.Attr into key-value pairs, applying the group prefix.
// Group-typed attributes are recursively flattened with dot-separated keys.
func appendAttr(fields []any, group string, a slog.Attr) []any {
	// Resolve LogValuer chains
	v := a.Value
	for {
		lv, ok := v.Any().(slog.LogValuer)
		if !ok {
			break
		}
		v = lv.LogValue()
	}

	a = slog.Attr{Key: a.Key, Value: v}
	if a.Equal(slog.Attr{}) {
		return fields
	}

	key := a.Key
	if group != "" {
		key = group + "." + key
	}

	if a.Value.Kind() == slog.KindGroup {
		for _, ga := range a.Value.Group() {
			fields = appendAttr(fields, key, ga)
		}
		return fields
	}
	return append(fields, key, a.Value.Any())
}
