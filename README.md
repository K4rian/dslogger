# dslogger
`dslogger` is a lightweight logger that uses zap for fast, structured logging and lumberjack for efficient log rotation.

## Usage
### Simple Console Logger
Use `NewSimpleConsoleLogger` to quickly create a console-only logger:
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
    // Create a console-only logger with default configuration at "info" level
    logger, err := dslogger.NewSimpleConsoleLogger("info")
    if err != nil {
        log.Fatalf("Error creating console logger: %v", err)
    }

    // Log an informational message
    logger.Info("This is an info message from the simple console logger!")
    logger.Warn("Unusual activity detected", "ip", "X.X.X.X")
    logger.Error("Error processing request", "requestID", "abc123")
}
```

### Simple Logger (Console and File)
Use `NewSimpleLogger` to create a logger that writes to both the console and a file (using default configuration):
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
    // Create a logger that outputs to both console and file at "debug" level
    logger, err := dslogger.NewSimpleLogger("debug")
    if err != nil {
        log.Fatalf("Error creating logger: %v", err)
    }

    // Log messages at various levels
    logger.Debug("Debug message", "user", "james")
    logger.Info("Info message", "action", "login")
    logger.Warn("Warning message", "ip", "X.X.X.X")
    logger.Error("Error message", "error", "file not found", "file", "/opt/files/myfile.txt")
}
```

### Service-Specific Logger
This example shows how to derive a service-specific logger from a base logger using the `WithService` method.<br>
The derived logger automatically attaches a `"service": "AuthService"` field to every log entry, making it easier to distinguish logs for different services in your application (especially when using JSON file format for logs):
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
    // Create a console-only logger with default configuration at "info" level
    logger, err := dslogger.NewSimpleConsoleLogger("info")
    if err != nil {
        log.Fatalf("Error creating console logger: %v", err)
    }
	logger.Debug("You shouldn't be able to see this message")
	logger.Info("A simple info message from the base logger!")

    // Derive a logger for a specific service
    authLogger := logger.WithService("AuthService")

    // Logs will include the "service" field when using JSON file format
    authLogger.Info("Authentication successful", "user", "james")
}
```

### Advanced Console-Only Logger
This example demonstrates how to create an advanced console-only logger with a custom configuration and options.<br>
It customizes the encoder settings and sets a custom service name using a functional option:
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
    // Define a custom configuration for console logging
    customConfig := &dslogger.Config{
        Level:                 "debug",
        ConsoleConfig:         dslogger.DefaultConsoleEncoderConfig,
        FieldSeparator:        ": ",
        ConsoleSeparator:      " | ",
        ServiceNameDecorators: [2]string{"[", "]"},
    }

    // Create a console-only logger with custom configuration and a functional option to set the service name
	logger, err := NewConsoleLogger("debug", customConfig, dslogger.WithServiceName("MyService"), dslogger.WithCallerSkip(1))
	if err != nil {
		log.Fatalf("Failed to create advanced console logger: %v", err)
	}

    logger.Info("This is a custom advanced console-only logger", "count", 14)
}
```

### Advanced Logger (Console and File)
Create an advanced logger with custom configuration and options that writes to both console and file.<br>
This example shows how to configure the file output format (JSON or Text) by setting the `LogFileFormat` and corresponding encoder configuration:
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
    // Define a custom configuration for the full logger
	customConfig := &dslogger.Config{
		LogFile:               "./full.log",
		LogFileFormat:         dslogger.LogFormatJSON, // Either LogFormatJSON or LogFormatText
		MaxSize:               50,
		MaxBackups:            7,
		MaxAge:                60,
		Compress:              true,
		Level:                 "debug",
		ConsoleConfig:         dslogger.DefaultConsoleEncoderConfig,
		FileConfig:            dslogger.DefaultJSONEncoderConfig, // For JSON output, use DefaultJSONEncoderConfig; for text, use DefaultTextEncoderConfig
		FieldSeparator:        "=",
		ConsoleSeparator:      "    ",
		ServiceNameDecorators: [2]string{"(", ")"},
	}

	// Create a logger that writes to both console and file, with a custom "branch" and "creation_time" fields
	logger, err := NewLogger("debug", customConfig, WithCustomFields("branch", "dev", "creation_time", time.Now()))
    if err != nil {
        log.Fatalf("Failed to create full logger: %v", err)
    }

    // Log messages at various levels.
    logger.Debug("Debug message with full logger", "user", "bob")
    logger.Info("Info message with full logger", "operation", "data_import")
    logger.Warn("Warning message", "warning", "low disk space")
    logger.Error("Error message", "error", "failed to connect to database")
}
```

## Installation
To install `dslogger`, simply run the following command:
```bash
go get github.com/k4rian/dslogger
```

For Go modules (if you're using Go 1.11+), ensure that your project is using modules and run:
```bash
go get github.com/k4rian/dslogger@latest
```

## License
[MIT][1]

[1]: https://github.com/K4rian/dslogger/blob/main/LICENSE
