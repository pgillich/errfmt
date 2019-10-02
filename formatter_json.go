package errorformatter

import (
	/*
		"fmt"
		"reflect"
		"strings"
	*/
	log "github.com/sirupsen/logrus"
)

/*
NewJSONLogger builds a customized Logrus JSON logger+formatter
	Features:
	* CallStackSkipLast
	* CallStackNewLines (only CallStackInFields)
	* ModuleCallerPrettyfier
*/
func NewJSONLogger(level log.Level, callStackSkipLast int, callStackNewLines bool,
) *AdvancedLogger {
	return &AdvancedLogger{
		Logger: &log.Logger{
			Formatter:    NewAdvancedJSONFormatter(),
			Level:        level,
			ReportCaller: true,
		},
		CallStackSkipLast: callStackSkipLast,
		CallStackNewLines: callStackNewLines,
	}
}

/*
AdvancedJSONFormatter is a customized Logrus JSON formatter
	Features:
	* ModuleCallerPrettyfier
*/
type AdvancedJSONFormatter struct {
	log.JSONFormatter
}

// NewAdvancedJSONFormatter makes a new AdvancedJSONFormatter
func NewAdvancedJSONFormatter() *AdvancedJSONFormatter {
	return &AdvancedJSONFormatter{
		JSONFormatter: log.JSONFormatter{
			CallerPrettyfier: ModuleCallerPrettyfier,
		},
	}
}

// Format implements logrus.Formatter interface
func (f *AdvancedJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	textPart, err := f.JSONFormatter.Format(entry)
	return textPart, err
}
