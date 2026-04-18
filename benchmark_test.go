package dslogger

import (
	"io"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// discardLogger builds a Logger whose console output goes to io.Discard
// so benchmarks measure formatting cost, not I/O.
func discardLogger(b *testing.B, level string) *Logger {
	b.Helper()
	cfg := NewDefaultConfig()
	cfg.NoColor = true

	atomicLvl := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if level == "debug" {
		atomicLvl = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	encoder := newDSConsoleEncoder(&cfg, cfg.ConsoleConfig, "")
	writer := zapcore.Lock(zapcore.AddSync(io.Discard))
	core := zapcore.NewCore(encoder, writer, atomicLvl)
	sugar := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)).Sugar()

	l := &Logger{
		config: &cfg,
		level:  atomicLvl,
	}
	l.consoleLogger.Store(sugar)
	return l
}

// BenchmarkInfo measures the cost of a single Info call with two key-value fields.
func BenchmarkInfo(b *testing.B) {
	logger := discardLogger(b, "info")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key1", "value1", "key2", 42)
	}
}

// BenchmarkInfoParallel measures Info throughput under contention.
func BenchmarkInfoParallel(b *testing.B) {
	logger := discardLogger(b, "info")
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("parallel message", "worker", 1)
		}
	})
}

// BenchmarkDisabledDebug verifies that Debug calls at INFO level are nearly zero-cost.
// The level gate should prevent any formatting or allocation.
func BenchmarkDisabledDebug(b *testing.B) {
	logger := discardLogger(b, "info")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("should be filtered", "key", "value")
	}
}

// BenchmarkWithFields measures the overhead of deriving a child logger with fields.
func BenchmarkWithFields(b *testing.B) {
	logger := discardLogger(b, "info")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		child := logger.WithFields("request_id", "req-123")
		child.Info("message")
	}
}

// BenchmarkRawZapBaseline provides a raw-zap comparison point for context.
func BenchmarkRawZapBaseline(b *testing.B) {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:    "T",
		LevelKey:   "L",
		MessageKey: "M",
		EncodeTime: zapcore.ISO8601TimeEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(io.Discard), zap.InfoLevel)
	sugar := zap.New(core).Sugar()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sugar.Infow("benchmark message", "key1", "value1", "key2", 42)
	}
	_ = os.Stdout // suppress unused import lint
}
