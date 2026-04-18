package dslogger

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var _pool = buffer.NewPool()

// kvPair holds a pre-formatted key-value pair accumulated by With() calls
type kvPair struct {
	key string
	val string
}

// dsConsoleEncoder is a custom zapcore.Encoder that formats console and text-file
// output with dslogger's ServiceNameDecorators, ConsoleSeparator, FieldSeparator,
// and log-injection sanitisation. It replaces the combination of zap's stock
// ConsoleEncoder + formatConsoleMessage.
type dsConsoleEncoder struct {
	cfg         *Config
	encCfg      zapcore.EncoderConfig
	serviceName string
	pairs       []kvPair // context fields from With()
	ns          string   // current namespace prefix (from OpenNamespace)
}

// newDSConsoleEncoder creates a dsConsoleEncoder.
func newDSConsoleEncoder(cfg *Config, encCfg zapcore.EncoderConfig, serviceName string) *dsConsoleEncoder {
	return &dsConsoleEncoder{
		cfg:         cfg,
		encCfg:      encCfg,
		serviceName: serviceName,
	}
}

// add appends a pre-formatted key-value pair, applying the current namespace prefix.
func (e *dsConsoleEncoder) add(key, val string) {
	e.pairs = append(e.pairs, kvPair{e.ns + key, val})
}

// ---------------------------------------------------------------------------
// ObjectEncoder (With)
func (e *dsConsoleEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	arr := &sliceArrayEncoder{}
	if err := v.MarshalLogArray(arr); err != nil {
		return err
	}
	e.add(key, "["+strings.Join(arr.elems, ", ")+"]")
	return nil
}

func (e *dsConsoleEncoder) AddObject(key string, v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	if err := v.MarshalLogObject(m); err != nil {
		return err
	}
	e.add(key, fmt.Sprint(m.Fields))
	return nil
}

func (e *dsConsoleEncoder) AddBinary(k string, v []byte) {
	e.add(k, fmt.Sprint(v))
}

func (e *dsConsoleEncoder) AddByteString(k string, v []byte) {
	e.add(k, string(v))
}

func (e *dsConsoleEncoder) AddBool(k string, v bool) {
	e.add(k, strconv.FormatBool(v))
}

func (e *dsConsoleEncoder) AddComplex128(k string, v complex128) {
	e.add(k, fmt.Sprint(v))
}

func (e *dsConsoleEncoder) AddComplex64(k string, v complex64) {
	e.add(k, fmt.Sprint(v))
}

func (e *dsConsoleEncoder) AddDuration(k string, v time.Duration) {
	e.add(k, v.String())
}

func (e *dsConsoleEncoder) AddFloat64(k string, v float64) {
	e.add(k, strconv.FormatFloat(v, 'f', -1, 64))
}

func (e *dsConsoleEncoder) AddFloat32(k string, v float32) {
	e.add(k, strconv.FormatFloat(float64(v), 'f', -1, 32))
}

func (e *dsConsoleEncoder) AddInt(k string, v int) {
	e.add(k, strconv.Itoa(v))
}

func (e *dsConsoleEncoder) AddInt64(k string, v int64) {
	e.add(k, strconv.FormatInt(v, 10))
}

func (e *dsConsoleEncoder) AddInt32(k string, v int32) {
	e.add(k, strconv.FormatInt(int64(v), 10))
}

func (e *dsConsoleEncoder) AddInt16(k string, v int16) {
	e.add(k, strconv.FormatInt(int64(v), 10))
}

func (e *dsConsoleEncoder) AddInt8(k string, v int8) {
	e.add(k, strconv.FormatInt(int64(v), 10))
}

func (e *dsConsoleEncoder) AddString(k string, v string) {
	e.add(k, sanitizeLogString(v))
}

func (e *dsConsoleEncoder) AddTime(k string, v time.Time) {
	enc := &singleValueEncoder{}
	if e.encCfg.EncodeTime != nil {
		e.encCfg.EncodeTime(v, enc)
		e.add(k, enc.val)
	} else {
		e.add(k, v.String())
	}
}

func (e *dsConsoleEncoder) AddUint(k string, v uint) {
	e.add(k, strconv.FormatUint(uint64(v), 10))
}

func (e *dsConsoleEncoder) AddUint64(k string, v uint64) {
	e.add(k, strconv.FormatUint(v, 10))
}

func (e *dsConsoleEncoder) AddUint32(k string, v uint32) {
	e.add(k, strconv.FormatUint(uint64(v), 10))
}

func (e *dsConsoleEncoder) AddUint16(k string, v uint16) {
	e.add(k, strconv.FormatUint(uint64(v), 10))
}

func (e *dsConsoleEncoder) AddUint8(k string, v uint8) {
	e.add(k, strconv.FormatUint(uint64(v), 10))
}

func (e *dsConsoleEncoder) AddUintptr(k string, v uintptr) {
	e.add(k, strconv.FormatUint(uint64(v), 10))
}

func (e *dsConsoleEncoder) AddReflected(k string, v any) error {
	e.add(k, fmt.Sprint(v))
	return nil
}

func (e *dsConsoleEncoder) OpenNamespace(key string) {
	if e.ns != "" {
		e.ns += key + "."
	} else {
		e.ns = key + "."
	}
}

// ---------------------------------------------------------------------------
// Encoder
func (e *dsConsoleEncoder) Clone() zapcore.Encoder {
	return &dsConsoleEncoder{
		cfg:         e.cfg,
		encCfg:      e.encCfg,
		serviceName: e.serviceName,
		pairs:       slices.Clone(e.pairs),
		ns:          e.ns,
	}
}

// EncodeEntry formats one log entry.
// Output format:
//
//	TIMESTAMP<sep>LEVEL<sep>CALLER<sep>[SERVICE] MESSAGE<sep>k1<fs>v1<sep>k2<fs>v2\n
//
// where <sep> = ConsoleSeparator and <fs> = FieldSeparator.
func (e *dsConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := _pool.Get()
	sep := e.cfg.ConsoleSeparator
	needsSep := false

	// Timestamp
	if e.encCfg.TimeKey != "" && e.encCfg.EncodeTime != nil {
		enc := &singleValueEncoder{}
		e.encCfg.EncodeTime(entry.Time, enc)
		buf.AppendString(enc.val)
		needsSep = true
	}

	// Level
	if e.encCfg.LevelKey != "" && e.encCfg.EncodeLevel != nil {
		if needsSep {
			buf.AppendString(sep)
		}
		enc := &singleValueEncoder{}
		e.encCfg.EncodeLevel(entry.Level, enc)
		buf.AppendString(enc.val)
		needsSep = true
	}

	// Caller
	if entry.Caller.Defined && e.encCfg.CallerKey != "" && e.encCfg.EncodeCaller != nil {
		if needsSep {
			buf.AppendString(sep)
		}
		enc := &singleValueEncoder{}
		e.encCfg.EncodeCaller(entry.Caller, enc)
		buf.AppendString(enc.val)
		needsSep = true
	}

	// Message (with optional service-name prefix)
	if needsSep {
		buf.AppendString(sep)
	}
	if e.serviceName != "" {
		buf.AppendString(e.cfg.ServiceNameDecorators[0])
		buf.AppendString(e.serviceName)
		buf.AppendString(e.cfg.ServiceNameDecorators[1])
		buf.AppendByte(' ')
	}
	buf.AppendString(sanitizeLogString(entry.Message))

	// Fields: context (from With) then per-call
	fieldSep := e.cfg.FieldSeparator
	for _, p := range e.pairs {
		buf.AppendString(sep)
		buf.AppendString(p.key)
		buf.AppendString(fieldSep)
		buf.AppendString(p.val)
	}
	for _, f := range fields {
		if f.Type == zapcore.SkipType {
			continue
		}
		buf.AppendString(sep)
		buf.AppendString(f.Key)
		buf.AppendString(fieldSep)
		buf.AppendString(formatField(f))
	}

	// Stack trace
	if entry.Stack != "" {
		buf.AppendByte('\n')
		buf.AppendString(entry.Stack)
	}

	buf.AppendByte('\n')
	return buf, nil
}

// formatField renders a zapcore.Field value as a console-friendly string
func formatField(f zapcore.Field) string {
	switch f.Type {
	case zapcore.ArrayMarshalerType:
		arr := &sliceArrayEncoder{}
		if m, ok := f.Interface.(zapcore.ArrayMarshaler); ok {
			_ = m.MarshalLogArray(arr)
		}
		return "[" + strings.Join(arr.elems, ", ") + "]"
	case zapcore.ObjectMarshalerType:
		m := zapcore.NewMapObjectEncoder()
		if obj, ok := f.Interface.(zapcore.ObjectMarshaler); ok {
			_ = obj.MarshalLogObject(m)
		}
		return fmt.Sprint(m.Fields)
	case zapcore.StringType:
		return sanitizeLogString(f.String)
	default:
		return fmt.Sprint(zapFieldValue(f))
	}
}

// ---------------------------------------------------------------------------
// Minimal PrimitiveArrayEncoder / ArrayEncoder implementations
// used to capture single values from EncodeTime / EncodeLevel / EncodeCaller
// and to format array-typed fields.

// singleValueEncoder captures the last value appended by a PrimitiveArrayEncoder callback.
type singleValueEncoder struct {
	val string
}

func (e *singleValueEncoder) AppendBool(v bool) {
	e.val = strconv.FormatBool(v)
}

func (e *singleValueEncoder) AppendByteString(v []byte) {
	e.val = string(v)
}

func (e *singleValueEncoder) AppendComplex128(v complex128) {
	e.val = fmt.Sprint(v)
}

func (e *singleValueEncoder) AppendComplex64(v complex64) {
	e.val = fmt.Sprint(v)
}

func (e *singleValueEncoder) AppendFloat64(v float64) {
	e.val = strconv.FormatFloat(v, 'f', -1, 64)
}
func (e *singleValueEncoder) AppendFloat32(v float32) {
	e.val = strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func (e *singleValueEncoder) AppendInt(v int) {
	e.val = strconv.Itoa(v)
}

func (e *singleValueEncoder) AppendInt64(v int64) {
	e.val = strconv.FormatInt(v, 10)
}

func (e *singleValueEncoder) AppendInt32(v int32) {
	e.val = strconv.FormatInt(int64(v), 10)
}

func (e *singleValueEncoder) AppendInt16(v int16) {
	e.val = strconv.FormatInt(int64(v), 10)
}

func (e *singleValueEncoder) AppendInt8(v int8) {
	e.val = strconv.FormatInt(int64(v), 10)
}

func (e *singleValueEncoder) AppendString(v string) {
	e.val = v
}

func (e *singleValueEncoder) AppendUint(v uint) {
	e.val = strconv.FormatUint(uint64(v), 10)
}

func (e *singleValueEncoder) AppendUint64(v uint64) {
	e.val = strconv.FormatUint(v, 10)
}

func (e *singleValueEncoder) AppendUint32(v uint32) {
	e.val = strconv.FormatUint(uint64(v), 10)
}

func (e *singleValueEncoder) AppendUint16(v uint16) {
	e.val = strconv.FormatUint(uint64(v), 10)
}

func (e *singleValueEncoder) AppendUint8(v uint8) {
	e.val = strconv.FormatUint(uint64(v), 10)
}

func (e *singleValueEncoder) AppendUintptr(v uintptr) {
	e.val = strconv.FormatUint(uint64(v), 10)
}

// sliceArrayEncoder accumulates formatted elements for AddArray / MarshalLogArray.
type sliceArrayEncoder struct {
	elems []string
}

func (e *sliceArrayEncoder) AppendBool(v bool) {
	e.elems = append(e.elems, strconv.FormatBool(v))
}

func (e *sliceArrayEncoder) AppendByteString(v []byte) {
	e.elems = append(e.elems, string(v))
}

func (e *sliceArrayEncoder) AppendComplex128(v complex128) {
	e.elems = append(e.elems, fmt.Sprint(v))
}

func (e *sliceArrayEncoder) AppendComplex64(v complex64) {
	e.elems = append(e.elems, fmt.Sprint(v))
}

func (e *sliceArrayEncoder) AppendFloat64(v float64) {
	e.elems = append(e.elems, strconv.FormatFloat(v, 'f', -1, 64))
}

func (e *sliceArrayEncoder) AppendFloat32(v float32) {
	e.elems = append(e.elems, strconv.FormatFloat(float64(v), 'f', -1, 32))
}

func (e *sliceArrayEncoder) AppendInt(v int) {
	e.elems = append(e.elems, strconv.Itoa(v))
}

func (e *sliceArrayEncoder) AppendInt64(v int64) {
	e.elems = append(e.elems, strconv.FormatInt(v, 10))
}

func (e *sliceArrayEncoder) AppendInt32(v int32) {
	e.elems = append(e.elems, strconv.FormatInt(int64(v), 10))
}

func (e *sliceArrayEncoder) AppendInt16(v int16) {
	e.elems = append(e.elems, strconv.FormatInt(int64(v), 10))
}

func (e *sliceArrayEncoder) AppendInt8(v int8) {
	e.elems = append(e.elems, strconv.FormatInt(int64(v), 10))
}

func (e *sliceArrayEncoder) AppendString(v string) {
	e.elems = append(e.elems, v)
}

func (e *sliceArrayEncoder) AppendUint(v uint) {
	e.elems = append(e.elems, strconv.FormatUint(uint64(v), 10))
}

func (e *sliceArrayEncoder) AppendUint64(v uint64) {
	e.elems = append(e.elems, strconv.FormatUint(v, 10))
}

func (e *sliceArrayEncoder) AppendUint32(v uint32) {
	e.elems = append(e.elems, strconv.FormatUint(uint64(v), 10))
}

func (e *sliceArrayEncoder) AppendUint16(v uint16) {
	e.elems = append(e.elems, strconv.FormatUint(uint64(v), 10))
}

func (e *sliceArrayEncoder) AppendUint8(v uint8) {
	e.elems = append(e.elems, strconv.FormatUint(uint64(v), 10))
}

func (e *sliceArrayEncoder) AppendUintptr(v uintptr) {
	e.elems = append(e.elems, strconv.FormatUint(uint64(v), 10))
}

func (e *sliceArrayEncoder) AppendDuration(v time.Duration) {
	e.elems = append(e.elems, v.String())
}

func (e *sliceArrayEncoder) AppendTime(v time.Time) {
	e.elems = append(e.elems, v.String())
}

func (e *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	inner := &sliceArrayEncoder{}
	if err := v.MarshalLogArray(inner); err != nil {
		return err
	}
	e.elems = append(e.elems, "["+strings.Join(inner.elems, ", ")+"]")
	return nil
}

func (e *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	if err := v.MarshalLogObject(m); err != nil {
		return err
	}
	e.elems = append(e.elems, fmt.Sprint(m.Fields))
	return nil
}

func (e *sliceArrayEncoder) AppendReflected(v any) error {
	e.elems = append(e.elems, fmt.Sprint(v))
	return nil
}
