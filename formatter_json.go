package errorformatter

import (
	log "github.com/sirupsen/logrus"
)

/*
NewJSONLogger builds a customized Logrus JSON logger+formatter
	Features:
	* CallStackSkipLast
	* CallStackNewLines (only CallStackInFields)
	* ModuleCallerPrettyfier
*/
func NewJSONLogger(level log.Level, flags int, callStackSkipLast int,
) *log.Logger {
	return &log.Logger{
		Formatter:    NewAdvancedJSONFormatter(flags, callStackSkipLast),
		Level:        level,
		ReportCaller: true,
	}
}

/*
AdvancedJSONFormatter is a customized Logrus JSON formatter
	Features:
	* ModuleCallerPrettyfier
*/
type AdvancedJSONFormatter struct {
	log.JSONFormatter
	AdvancedFormatter
}

// NewAdvancedJSONFormatter makes a new AdvancedJSONFormatter
func NewAdvancedJSONFormatter(flags int, callStackSkipLast int) *AdvancedJSONFormatter {
	return &AdvancedJSONFormatter{
		JSONFormatter: log.JSONFormatter{
			CallerPrettyfier: ModuleCallerPrettyfier,
		},
		AdvancedFormatter: AdvancedFormatter{
			Flags:             flags,
			CallStackSkipLast: callStackSkipLast,
		},
	}
}

// Format implements logrus.Formatter interface
func (f *AdvancedJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	f.FillDetailsToFields(entry)
	callStackLines := f.FillCallStack(entry)

	textPart, err := f.JSONFormatter.Format(entry)

	textPart = f.AppendCallStack(textPart, callStackLines)

	return textPart, err
}
