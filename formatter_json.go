package errfmt

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
	logger := log.New()

	logger.Formatter = NewAdvancedJSONFormatter(flags, callStackSkipLast)
	logger.Level = level
	logger.ReportCaller = true

	return logger
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
	entry.Data = f.MergeDetailsToFields(entry)
	callStackLines := f.GetCallStack(entry)

	if (f.Flags & FlagCallStackInFields) > 0 {
		entry.Data[KeyCallStack] = callStackLines
	}

	textPart, err := f.JSONFormatter.Format(entry)

	if (f.Flags & FlagCallStackOnConsole) > 0 {
		textPart = f.AppendCallStack(textPart, callStackLines)
	}

	return textPart, err
}
