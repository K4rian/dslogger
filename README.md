# dslogger

## Overview
`dslogger` is a lightweight logger that uses zap for fast, structured logging and lumberjack for efficient log rotation.

## Usage
```go
package main

import (
  "github.com/K4rian/dslogger"
)

func main() {
	// Create a simple console logger with the default log level "info".
	consoleLogger := dslogger.NewSimpleConsoleLogger("info")
	consoleLogger.Info("This is an informative message")
	consoleLogger.Warn("Warning! Something might be wrong")
	consoleLogger.Error("An error occurred", "error", "sample error")

	// Derive a new logger for the "auth-service" using the console logger.
	authLogger := consoleLogger.WithService("auth-service")
	authLogger.Info("Authentication service started", "version", "1.0.0")

	// Create a file logger with a log level of "debug" and custom configuration.
	fileLogger := dslogger.NewLogger("debug", &dslogger.Config{
		LogFile:       "./logfile.log",
		LogFileFormat: dslogger.LogFormatText, // or dslogger.LogFormatJSON
	})
	fileLogger.Info("You shouldn't see this info message at debug level")
	fileLogger.Debug("This debug message will be appended to the log file!")

  // Change the file logger's level to "warn" so that only warnings and errors are logged.
	fileLogger.SetLogLevel("warn")
	fileLogger.Info("This info message will not be logged at the 'warn' level")
	fileLogger.Debug("Nor will this debug message")
	fileLogger.Warn("Failed to save to the file", "reason", "permission denied")
}
```

## Install
To install dslogger, simply run the following command:
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
