package dslogger

import "fmt"

func formatConsoleFields(fieldSeparator string, consoleSeparator string, fields ...interface{}) string {
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

func (l *Logger) formatMessage(msg string, fields ...interface{}) string {
	if l.serviceName != "" {
		msg = fmt.Sprintf("%s%s%s %s",
			l.config.ServiceNameDecorators[0],
			l.serviceName,
			l.config.ServiceNameDecorators[1],
			msg)
	}

	if len(fields) > 0 {
		msg += l.config.ConsoleSeparator + formatConsoleFields(l.config.FieldSeparator, l.config.ConsoleSeparator, fields...)
	}
	return msg
}
