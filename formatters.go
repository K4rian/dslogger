package dslogger

import "fmt"

func formatConsoleFields(separator string, fields ...interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	formatted := ""
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			formatted += fmt.Sprintf("%v%s%v ", fields[i], separator, fields[i+1])
		}
	}
	if len(formatted) > 0 {
		formatted = formatted[:len(formatted)-1]
	}
	return formatted
}

func (l *Logger) formatMessage(msg string, fields ...interface{}) string {
	if l.serviceName != "" {
		msg = fmt.Sprintf("%s%s%s %s",
			l.config.ServiceNameDecorators[0],
			l.serviceName,
			l.config.ServiceNameDecorators[1],
			msg)
	}

	if len(fields) > 0 {
		msg += l.config.ConsoleSeparator + formatConsoleFields(l.config.FieldSeparator, fields...)
	}
	return msg
}
