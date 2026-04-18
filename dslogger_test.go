package dslogger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

// withTempLogFile runs fn with a Config.LogFile pointing at a fresh temp path.
func withTempLogFile(t *testing.T, fn func(path string, cfg *Config)) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	cfg := NewDefaultConfig()
	cfg.LogFile = path
	fn(path, &cfg)
}

// TestConfigNotMutated checks that newLogger does not mutate the caller's Config.
func TestConfigNotMutated(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.LogFile = "snapshot.log"
	cfg.ConsoleSeparator = " | "
	// Capture the original encoder config, specifically,
	// note whether EncodeLevel is nil
	origConsoleSep := cfg.ConsoleSeparator
	origLogFile := cfg.LogFile
	origLevelFormats := cfg.LevelFormats

	_, err := NewConsoleLogger("info", &cfg)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LogFile != origLogFile {
		t.Errorf("LogFile mutated: got %q want %q", cfg.LogFile, origLogFile)
	}
	if cfg.ConsoleSeparator != origConsoleSep {
		t.Errorf("ConsoleSeparator mutated: got %q want %q", cfg.ConsoleSeparator, origConsoleSep)
	}
	// Same map pointer, mutation would otherwise propagate back
	if len(cfg.LevelFormats) != len(origLevelFormats) {
		t.Errorf("LevelFormats map mutated")
	}
}

// TestDefaultConfigIsolation verifies that two independently-constructed default
// configs do not share any map backing, mutating one must not affect the other.
func TestDefaultConfigIsolation(t *testing.T) {
	a := NewDefaultConfig()
	b := NewDefaultConfig()

	a.LevelFormats[zapcore.InfoLevel] = LevelFormat{LevelStr: "CUSTOM"}

	if b.LevelFormats[zapcore.InfoLevel].LevelStr == "CUSTOM" {
		t.Errorf("DefaultConfig LevelFormats shared across instances")
	}
}

// TestNewSimpleLoggerDoesNotMutateGlobal ensures repeated Simple constructor
// calls do not mutate the package-level DefaultLoggerConfig.
func TestNewSimpleLoggerDoesNotMutateGlobal(t *testing.T) {
	origLevelFormats := make(map[zapcore.Level]LevelFormat, len(DefaultLoggerConfig.LevelFormats))
	for k, v := range DefaultLoggerConfig.LevelFormats {
		origLevelFormats[k] = v
	}

	for i := 0; i < 5; i++ {
		_, err := NewSimpleConsoleLogger("info")
		if err != nil {
			t.Fatal(err)
		}
	}

	for k, v := range origLevelFormats {
		if DefaultLoggerConfig.LevelFormats[k] != v {
			t.Errorf("DefaultLoggerConfig.LevelFormats mutated at %v: got %v want %v",
				k, DefaultLoggerConfig.LevelFormats[k], v)
		}
	}
}

// TestConcurrentLoggingWithLevelChanges exercises the hot path against concurrent
// SetLogLevel calls.
// This used to race under the previous core-rebuild model.
func TestConcurrentLoggingWithLevelChanges(t *testing.T) {
	withTempLogFile(t, func(_ string, cfg *Config) {
		logger, err := NewLogger("debug", cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer logger.Close()

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 200; j++ {
					logger.Info("concurrent log", "worker", id, "iter", j)
					logger.WithFields("k", "v").Debug("derived")
				}
			}(i)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = logger.SetLogLevel("info")
				_ = logger.SetLogLevel("debug")
			}
		}()
		wg.Wait()
	})
}

// TestWithFieldsNoAliasing ensures that sibling loggers derived from the same
// parent do not share slice backing, a classic append-on-shared-slice bug.
func TestWithFieldsNoAliasing(t *testing.T) {
	logger, err := NewSimpleConsoleLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	base := logger.WithFields("a", 1)
	child1 := base.WithFields("b", 2)
	child2 := base.WithFields("c", 3)

	if len(child1.Fields()) != 2 {
		t.Errorf("child1 fields = %d, want 2", len(child1.Fields()))
	}
	if len(child2.Fields()) != 2 {
		t.Errorf("child2 fields = %d, want 2", len(child2.Fields()))
	}

	for _, f := range child1.Fields() {
		if f.Key == "c" {
			t.Errorf("child1 aliased with child2's field")
		}
	}
	for _, f := range child2.Fields() {
		if f.Key == "b" {
			t.Errorf("child2 aliased with child1's field")
		}
	}
}

// TestOddFieldCountDoesNotPanic verifies that WithFields / Info tolerate odd
// field counts instead of panicking.
func TestOddFieldCountDoesNotPanic(t *testing.T) {
	logger, err := NewSimpleConsoleLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	_ = logger.WithFields("lonely")
	logger.Info("msg", "only-key")
}

// TestSetLogLevelInvalidRejects verifies that bad level strings now return an error.
func TestSetLogLevelInvalidRejects(t *testing.T) {
	logger, err := NewSimpleConsoleLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.SetLogLevel("bogus"); err == nil {
		t.Error("SetLogLevel should reject bogus level")
	}
}

// TestLevelGatingBeforeFormatting: at INFO, calling Debug on a sub-logger must
// not evaluate formatConsoleMessage.
func TestLevelGatingSuppressesDebug(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer logger.Close()

		logger.Debug("should not appear", "k", "v")
		logger.Info("should appear", "k", "v")
		_ = logger.Sync()

		data, _ := os.ReadFile(path)
		out := string(data)
		if strings.Contains(out, "should not appear") {
			t.Errorf("Debug line written despite level=info: %q", out)
		}
		if !strings.Contains(out, "should appear") {
			t.Errorf("Info line missing: %q", out)
		}
	})
}

// TestJSONFileFormatWithCustomFields verifies that WithCustomFields values
// reach the JSON file output.
func TestJSONFileFormatWithCustomFields(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatJSON
		cfg.FileConfig = DefaultJSONEncoderConfig
		logger, err := NewLogger("info", cfg, WithCustomFields("app", "demo"))
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("hello")
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(path)
		if len(data) == 0 {
			t.Fatal("no JSON output")
		}
		lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
		var parsed map[string]any
		if err := json.Unmarshal(lines[0], &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nraw: %s", err, lines[0])
		}
		if parsed["app"] != "demo" {
			t.Errorf("custom field missing from JSON: got %v", parsed)
		}
	})
}

// TestJSONFileFormatWithDerivedFields verifies that WithFields values reach
// the JSON file output too, the old API split made them console-only.
func TestJSONFileFormatWithDerivedFields(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatJSON
		cfg.FileConfig = DefaultJSONEncoderConfig
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		derived := logger.WithFields("request_id", "req-42")
		derived.Info("hello")
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(path)
		lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
		var parsed map[string]any
		if err := json.Unmarshal(lines[len(lines)-1], &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["request_id"] != "req-42" {
			t.Errorf("WithFields value missing from JSON: %v", parsed)
		}
	})
}

// TestSyncFlushes ensures buffered log lines actually reach disk after Close.
func TestSyncFlushes(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("persisted")
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(path)
		if !bytes.Contains(data, []byte("persisted")) {
			t.Errorf("Close did not flush buffered output: %q", data)
		}
	})
}

// TestWithContextTypedKeys verifies that only typed context keys are consulted.
func TestWithContextTypedKeys(t *testing.T) {
	logger, err := NewSimpleConsoleLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	// String-keyed value should NOT be picked up (avoids package collisions)
	ctx := context.WithValue(context.Background(), "request_id", "should-be-ignored")
	derived := logger.WithContext(ctx)
	for _, f := range derived.Fields() {
		if f.Key == string(RequestIDKey) {
			t.Errorf("string-keyed context value leaked into fields: %v", f)
		}
	}

	// Typed-key value should be picked up
	ctx2 := context.WithValue(context.Background(), RequestIDKey, "req-7")
	derived2 := logger.WithContext(ctx2)
	found := false
	for _, f := range derived2.Fields() {
		if f.Key == string(RequestIDKey) {
			found = true
		}
	}
	if !found {
		t.Errorf("typed-key context value not picked up")
	}
}

// TestLogInjectionSanitized verifies newline-injection in messages is escaped.
func TestLogInjectionSanitized(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("safe\ninjected INFO fake line")
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		if bytes.Count(data, []byte("\n")) != 1 {
			t.Errorf("unexpected newline count in output: %q", data)
		}
		if !bytes.Contains(data, []byte(`safe\ninjected`)) {
			t.Errorf("newline not escaped: %q", data)
		}
	})
}

// TestCallerInfoPointsAtCaller parses the file output and asserts that the reported
// caller file is this test file.
// Guards against future inliner or caller-skip changes.
func TestCallerInfoPointsAtCaller(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.FileConfig.CallerKey = "caller"
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("caller check") // <- the call site
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		line := string(data)
		// The caller should reference dslogger_test.go, not logger.go or options.go.
		if !strings.Contains(line, "dslogger_test.go:") {
			t.Errorf("caller does not point at test file: %q", line)
		}
	})
}

// TestGoldenTextFormat locks in the human-readable console/text format.
func TestGoldenTextFormat(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatText
		cfg.NoColor = true
		cfg.FileConfig.EncodeTime = func(t2 time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("FIXED_TIME")
		}
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("hello world", "key1", "val1", "key2", 42)
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		got := string(data)
		want := "FIXED_TIME | INFO  | hello world | key1: val1 | key2: 42\n"
		if got != want {
			t.Errorf("golden text format mismatch:\ngot:  %q\nwant: %q", got, want)
		}
	})
}

// TestGoldenTextFormatWithService verifies service name decorators in text output.
func TestGoldenTextFormatWithService(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatText
		cfg.NoColor = true
		cfg.FileConfig.EncodeTime = func(t2 time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("FIXED_TIME")
		}
		logger, err := NewLogger("info", cfg, WithServiceName("myapp"))
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("started", "port", 8080)
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		got := string(data)
		want := "FIXED_TIME | INFO  | [myapp] started | port: 8080\n"
		if got != want {
			t.Errorf("golden text+service format mismatch:\ngot:  %q\nwant: %q", got, want)
		}
	})
}

// TestGoldenJSONFormat locks in the JSON file format contract.
func TestGoldenJSONFormat(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatJSON
		cfg.FileConfig = DefaultJSONEncoderConfig
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("hello", "env", "prod")
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
		var parsed map[string]any
		if err := json.Unmarshal(lines[0], &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nraw: %s", err, lines[0])
		}
		// Check required keys
		for _, key := range []string{"timestamp", "level", "message", "env"} {
			if _, ok := parsed[key]; !ok {
				t.Errorf("missing key %q in JSON output: %v", key, parsed)
			}
		}
		if parsed["message"] != "hello" {
			t.Errorf("message mismatch: got %v", parsed["message"])
		}
		if parsed["env"] != "prod" {
			t.Errorf("field mismatch: got %v", parsed["env"])
		}
		if parsed["level"] != "INFO" {
			t.Errorf("level mismatch: got %v", parsed["level"])
		}
	})
}

// TestForceColorOverridesAutoDetect verifies that ForceColor=true keeps ANSI
// output even when NoColor would otherwise be auto-set.
func TestForceColorOverridesAutoDetect(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.ForceColor = true
	cfg.NoColor = false
	cloned := cloneConfig(&cfg)
	applyDefaults(cloned)
	if cloned.NoColor {
		t.Errorf("ForceColor=true did not prevent NoColor auto-detection")
	}
}

// TestConfigLevelFallback verifies that an empty constructor-level argument
// falls back to Config.Level.
func TestConfigLevelFallback(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Level = "warn"
	logger, err := NewConsoleLogger("", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if logger.Level() != zapcore.WarnLevel {
		t.Errorf("expected warn level from Config.Level fallback, got %v", logger.Level())
	}
}

// TestSlogHandlerIntegration verifies that the slog.Handler bridge routes
// messages through the dslogger Logger and respects level gating.
func TestSlogHandlerIntegration(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatJSON
		cfg.FileConfig = DefaultJSONEncoderConfig
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		slogger := slog.New(NewSlogHandler(logger))

		slogger.Debug("should not appear")
		slogger.Info("hello from slog", "count", 42)
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "should not appear") {
			t.Error("debug message appeared despite info level")
		}

		lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		var parsed map[string]any
		if err := json.Unmarshal(lines[0], &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["message"] != "hello from slog" {
			t.Errorf("message mismatch: %v", parsed["message"])
		}
		if parsed["count"] != float64(42) {
			t.Errorf("field mismatch: %v", parsed["count"])
		}
	})
}

// TestSlogHandlerWithGroup verifies that WithGroup prefixes attribute keys.
func TestSlogHandlerWithGroup(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.LogFileFormat = LogFormatJSON
		cfg.FileConfig = DefaultJSONEncoderConfig
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		slogger := slog.New(NewSlogHandler(logger)).WithGroup("http")
		slogger.Info("request", "method", "GET", "path", "/api")
		_ = logger.Close()

		data, _ := os.ReadFile(path)
		var parsed map[string]any
		if err := json.Unmarshal(bytes.TrimRight(data, "\n"), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["http.method"] != "GET" {
			t.Errorf("grouped field missing: %v", parsed)
		}
	})
}

// TestFileMode verifies that the log file is pre-created with the configured permissions.
func TestFileMode(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		cfg.FileMode = 0640
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}
		logger.Info("test")
		_ = logger.Close()

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		got := info.Mode().Perm()
		if got != 0640 {
			t.Errorf("file mode = %04o, want 0640", got)
		}
	})
}

// TestFatalWritesAndExits verifies Fatal logs and calls osExit.
func TestFatalWritesAndExits(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}

		var exitCode int
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = os.Exit }()

		logger.Fatal("goodbye", "reason", "test")
		_ = logger.Close()

		if exitCode != 1 {
			t.Errorf("osExit called with %d, want 1", exitCode)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "[FATAL] goodbye") {
			t.Errorf("fatal message missing from output: %q", data)
		}
	})
}

// TestPanicWritesAndPanics verifies Panic logs and panics.
func TestPanicWritesAndPanics(t *testing.T) {
	withTempLogFile(t, func(path string, cfg *Config) {
		logger, err := NewLogger("info", cfg)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic, got none")
			}
			if r != "boom" {
				t.Errorf("panic value = %v, want 'boom'", r)
			}
			_ = logger.Close()
			data, _ := os.ReadFile(path)
			if !strings.Contains(string(data), "[PANIC] boom") {
				t.Errorf("panic message missing from output: %q", data)
			}
		}()

		logger.Panic("boom", "key", "val")
	})
}

// TestConsoleWriter verifies that ConsoleWriter redirects console output.
func TestConsoleWriter(t *testing.T) {
	var buf bytes.Buffer
	cfg := NewDefaultConfig()
	cfg.ConsoleWriter = &buf
	cfg.NoColor = true

	logger, err := NewConsoleLogger("info", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("routed", "dest", "buffer")
	_ = logger.Sync()

	out := buf.String()
	if !strings.Contains(out, "routed") {
		t.Errorf("console output not routed to custom writer: %q", out)
	}
	if !strings.Contains(out, "dest: buffer") {
		t.Errorf("fields missing from custom writer output: %q", out)
	}
}

// TestConsoleFormatJSON verifies structured JSON output on the console path.
func TestConsoleFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := NewDefaultConfig()
	cfg.ConsoleWriter = &buf
	cfg.ConsoleFormat = LogFormatJSON
	cfg.ConsoleConfig = DefaultJSONEncoderConfig

	logger, err := NewConsoleLogger("info", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello", "env", "test")
	_ = logger.Sync()

	var parsed map[string]any
	if err := json.Unmarshal(bytes.TrimRight(buf.Bytes(), "\n"), &parsed); err != nil {
		t.Fatalf("invalid JSON from console: %v\nraw: %s", err, buf.String())
	}
	if parsed["message"] != "hello" {
		t.Errorf("message mismatch: %v", parsed)
	}
	if parsed["env"] != "test" {
		t.Errorf("field mismatch: %v", parsed)
	}
}

// TestWithContextOTel verifies that WithContext extracts OpenTelemetry trace/span IDs.
func TestWithContextOTel(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("4f9c2a7b1e6d8c3f0a2b4c6d8e1f9a0b")
	spanID, _ := trace.SpanIDFromHex("1a2b3c4d5e6f7a8b")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	logger, err := NewSimpleConsoleLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	derived := logger.WithContext(ctx)

	foundTrace := false
	foundSpan := false
	for _, f := range derived.Fields() {
		if f.Key == string(TraceIDKey) {
			foundTrace = true
		}
		if f.Key == string(SpanIDKey) {
			foundSpan = true
		}
	}
	if !foundTrace {
		t.Error("trace_id not extracted from OTel span context")
	}
	if !foundSpan {
		t.Error("span_id not extracted from OTel span context")
	}
}
