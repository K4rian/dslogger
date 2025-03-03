package dslogger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// initializeLogFuncMaps sets up the logging function maps for both console and file outputs.
func initializeLogFuncMaps(l *Logger) {
	l.logConsoleFuncs = map[string]func(...any){
		zapcore.InfoLevel.String():  l.consoleLogger.Info,
		zapcore.DebugLevel.String(): l.consoleLogger.Debug,
		zapcore.WarnLevel.String():  l.consoleLogger.Warn,
		zapcore.ErrorLevel.String(): l.consoleLogger.Error,
	}

	if l.fileLogger != nil {
		if l.config.LogFileFormat == LogFormatText {
			l.logTextFuncs = map[string]func(...any){
				zapcore.InfoLevel.String():  l.fileLogger.Info,
				zapcore.DebugLevel.String(): l.fileLogger.Debug,
				zapcore.WarnLevel.String():  l.fileLogger.Warn,
				zapcore.ErrorLevel.String(): l.fileLogger.Error,
			}
		} else {
			l.logJSONFuncs = map[string]func(string, ...any){
				zapcore.InfoLevel.String():  l.fileLogger.Infow,
				zapcore.DebugLevel.String(): l.fileLogger.Debugw,
				zapcore.WarnLevel.String():  l.fileLogger.Warnw,
				zapcore.ErrorLevel.String(): l.fileLogger.Errorw,
			}
		}
	}
}

// isLogFuncMapsInitialized checks if the logging function maps for console and file are initialized.
func isLogFuncMapsInitialized(l *Logger) bool {
	return l.logConsoleFuncs != nil &&
		(l.fileLogger == nil || l.logTextFuncs != nil) &&
		(l.fileLogger == nil || l.config.LogFileFormat != LogFormatJSON || l.logJSONFuncs != nil)
}

// zapFieldsToAny converts a slice of zap.Field to a slice of interface{}
// by extracting each field's key and an appropriate Go value based on its type.
func zapFieldsToAny(fields []zap.Field) []any {
	ret := []any{}
	for _, f := range fields {
		var value any
		switch f.Type {
		case zapcore.UnknownType:
			value = f.String
		case zapcore.ArrayMarshalerType, zapcore.ObjectMarshalerType, zapcore.StringerType, zapcore.ErrorType, zapcore.InlineMarshalerType:
			value = f.Interface
		case zapcore.BinaryType, zapcore.ByteStringType:
			value = f.String
		case zapcore.BoolType:
			value = f.Integer != 0
		case zapcore.Complex128Type, zapcore.Complex64Type:
			value = f.Interface
		case zapcore.DurationType:
			value = time.Duration(f.Integer)
		case zapcore.Float64Type, zapcore.Float32Type:
			value = f.Interface
		case zapcore.Int64Type:
			value = f.Integer
		case zapcore.Int32Type:
			value = int32(f.Integer)
		case zapcore.Int16Type:
			value = int16(f.Integer)
		case zapcore.Int8Type:
			value = int8(f.Integer)
		case zapcore.StringType:
			value = f.String
		case zapcore.TimeType, zapcore.TimeFullType:
			if t, ok := f.Interface.(time.Time); ok {
				value = t.Format(time.RFC3339Nano)
			} else if loc, ok := f.Interface.(*time.Location); ok {
				value = time.Now().In(loc).Format(time.RFC3339Nano)
			} else {
				value = f.String
			}
		case zapcore.Uint64Type:
			value = uint64(f.Integer)
		case zapcore.Uint32Type:
			value = uint32(f.Integer)
		case zapcore.Uint16Type:
			value = uint16(f.Integer)
		case zapcore.Uint8Type:
			value = uint8(f.Integer)
		case zapcore.UintptrType:
			value = uintptr(f.Integer)
		case zapcore.ReflectType:
			value = f.Interface
		case zapcore.NamespaceType: // no-op field
			value = f.String
		case zapcore.SkipType:
			continue
		default:
			value = f.String
		}
		ret = append(ret, f.Key, value)
	}
	return ret
}
