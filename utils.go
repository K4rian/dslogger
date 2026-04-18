package dslogger

import (
	"math"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapFieldValue extracts the Go value from a zap.Field for console formatting.
// The type switch mirrors zap's internal Field representation:
// Float fields store bits in Integer, and Time fields store an epoch value
// in Integer plus a *time.Location in Interface.
func zapFieldValue(f zap.Field) any {
	switch f.Type {
	case zapcore.UnknownType:
		return f.String
	case zapcore.ArrayMarshalerType,
		zapcore.ObjectMarshalerType,
		zapcore.StringerType,
		zapcore.ErrorType,
		zapcore.InlineMarshalerType,
		zapcore.ReflectType:
		return f.Interface
	case zapcore.BinaryType, zapcore.ByteStringType:
		return f.String
	case zapcore.BoolType:
		return f.Integer != 0
	case zapcore.Complex128Type, zapcore.Complex64Type:
		return f.Interface
	case zapcore.DurationType:
		return time.Duration(f.Integer)
	case zapcore.Float64Type:
		return math.Float64frombits(uint64(f.Integer))
	case zapcore.Float32Type:
		return math.Float32frombits(uint32(f.Integer))
	case zapcore.Int64Type:
		return f.Integer
	case zapcore.Int32Type:
		return int32(f.Integer)
	case zapcore.Int16Type:
		return int16(f.Integer)
	case zapcore.Int8Type:
		return int8(f.Integer)
	case zapcore.StringType:
		return f.String
	case zapcore.TimeType:
		// zap.Time stores epoch nanos in Integer and *time.Location in Interface
		loc, _ := f.Interface.(*time.Location)
		if loc == nil {
			loc = time.UTC
		}
		return time.Unix(0, f.Integer).In(loc)
	case zapcore.TimeFullType:
		if t, ok := f.Interface.(time.Time); ok {
			return t
		}
		return time.Time{}
	case zapcore.Uint64Type:
		return uint64(f.Integer)
	case zapcore.Uint32Type:
		return uint32(f.Integer)
	case zapcore.Uint16Type:
		return uint16(f.Integer)
	case zapcore.Uint8Type:
		return uint8(f.Integer)
	case zapcore.UintptrType:
		return uintptr(f.Integer)
	case zapcore.NamespaceType:
		return ""
	default:
		return f.String
	}
}
