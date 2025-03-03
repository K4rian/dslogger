package dslogger

import (
	"fmt"
)

// formatConsoleMessage formats the log message by optionally decorating it with the service name and appending formatted fields.
// This method is typically used by the Logger to generate the final message string for console output.
func (l *Logger) formatConsoleMessage(msg string, fields ...any) string {
	// Prepend service name if available.
	if l.serviceName != "" {
		msg = fmt.Sprintf("%s%s%s %s",
			l.config.ServiceNameDecorators[0],
			l.serviceName,
			l.config.ServiceNameDecorators[1],
			msg,
		)
	}

	// Combine custom fields (stored in customFields) with any runtime fields
	var combined []any
	if len(l.customFields) > 0 {
		combined = append(zapFieldsToAny(l.customFields), fields...)
	} else {
		combined = fields
	}

	// If there are fields to append, use the configured ConsoleSeparator and FieldSeparator
	if len(combined) > 0 {
		// formatConsoleFields uses config.FieldSeparator to join key/value pairs,
		// and config.ConsoleSeparator to join each pair
		msg += l.config.ConsoleSeparator + formatConsoleFields(l.config.FieldSeparator, l.config.ConsoleSeparator, combined...)
	}
	return msg
}

// formatConsoleFields formats a list of fields into a single string using the provided separators.
// Fields are expected to be provided in key/value pairs.
func formatConsoleFields(fieldSeparator string, consoleSeparator string, fields ...any) string {
	if len(fields) == 0 {
		return ""
	}

	formatted := ""
	fieldsLen := len(fields)
	for i := 0; i < fieldsLen; i += 2 {
		if i+1 < fieldsLen {
			formatted += fmt.Sprintf("%v%s%v", fields[i], fieldSeparator, fields[i+1])
			if i+2 < fieldsLen {
				formatted += consoleSeparator
			}
		}
	}
	return formatted
}
