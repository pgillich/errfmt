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

	if flags&FlagExtractDetails > 0 {
		logger.AddHook(HookAllLevels(AppendDetailsToEntry))
	}

	if flags&FlagCallStackInFields > 0 {
		logger.AddHook(HookAllLevels(AppendCallStackToEntry(callStackSkipLast)))
	}

	if flags&FlagPrintStructFieldNames > 0 {
		logger.AddHook(HookAllLevels(RenderStructFieldNames))
	}

	return logger
}

/*
AdvancedJSONFormatter is a customized Logrus JSON formatter
	Features:
	* ModuleCallerPrettyfier
*/
type AdvancedJSONFormatter struct {
	log.JSONFormatter
	ConsoleFlags
	AdvancedFormatter
}

// NewAdvancedJSONFormatter makes a new AdvancedJSONFormatter
func NewAdvancedJSONFormatter(flags int, callStackSkipLast int) *AdvancedJSONFormatter {
	return &AdvancedJSONFormatter{
		JSONFormatter: log.JSONFormatter{
			CallerPrettyfier: ModuleCallerPrettyfier,
		},
		ConsoleFlags: ConsoleFlags{
			CallStackOnConsole: flags&FlagCallStackOnConsole > 0,
			CallStackSkipLast:  callStackSkipLast,
		},
		AdvancedFormatter: AdvancedFormatter{
			Flags:              flags,
			CallStackSkipLastX: callStackSkipLast,
		},
	}
}

// Format implements logrus.Formatter interface
// calls logrus.TextFormatter.Format()
func (f *AdvancedJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	var consoleCallStackLines []string

	if f.CallStackOnConsole {
		consoleCallStackLines = GetCallStack(entry)
	}

	// consoleCallStackLines cannot be dig anymore
	RenderErrorInEntry(entry)

	textPart, err := f.JSONFormatter.Format(entry)

	if len(consoleCallStackLines) > f.CallStackSkipLast {
		textPart = AppendCallStack(textPart, consoleCallStackLines[:len(consoleCallStackLines)-f.CallStackSkipLast])
	}

	return textPart, err
}
