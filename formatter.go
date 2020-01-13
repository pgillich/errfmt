package errfmt

import (
	"fmt"
	"reflect"
	"strings"

	"emperror.dev/errors"
	"emperror.dev/errors/utils/keyval"

	log "github.com/sirupsen/logrus"
)

// AdvancedFormatter contains the advanced formatting config
type AdvancedFormatter struct {
	// Flags is formatting flags
	Flags int
	// CallStackSkipLast skips the last lines
	CallStackSkipLastX int
}

type ConsoleFlags struct {
	// CallStackOnConsole extracts errors.StackTrace() to logrus.Field "callstack"
	CallStackOnConsole bool
	// CallStackSkipLast skips the last lines
	CallStackSkipLast int
	/*
		// CallStackInHTTPProblem extracts errors.StackTrace() to HTTPProblem
		CallStackInHTTPProblem bool
	*/
}

// GetError extracts error from entry.Data (a result of log.WithError())
func GetError(entry *log.Entry) error {
	if errVal, ok := entry.Data[log.ErrorKey]; ok {
		if err, ok := errVal.(error); ok {
			return err
		}
	}

	return nil
}

// PrepareFields copies entry.Data to new and prepares for formatting
func (f *AdvancedFormatter) PrepareFields(entry *log.Entry, fixedFields []string) log.Fields {
	data := f.MergeDetailsToFields(entry)

	for _, key := range fixedFields {
		prefixFieldClashes(data, key)
	}

	callStackLines := f.GetCallStack(entry)
	if (f.Flags & FlagCallStackInFields) > 0 { // nolint:wsl
		data[KeyCallStack] = callStackLines
	}

	f.RenderFieldValues(data)

	if entry.HasCaller() {
		funcVal, fileVal := ModuleCallerPrettyfier(entry.Caller)
		data[log.FieldKeyFunc] = funcVal
		data[log.FieldKeyFile] = fileVal
	}

	if entry.Level != log.PanicLevel {
		data[log.FieldKeyLevel] = entry.Level
	}

	return data
}

func PrepareFields(entry *log.Entry, fixedFields []string) {
	for _, key := range fixedFields {
		prefixFieldClashes(entry.Data, key)
	}

	if entry.HasCaller() {
		funcVal, fileVal := ModuleCallerPrettyfier(entry.Caller)
		entry.Data[log.FieldKeyFunc] = funcVal
		entry.Data[log.FieldKeyFile] = fileVal
	}

	entry.Data[log.FieldKeyLevel] = entry.Level
}

// MergeDetailsToFields merges Details from error to, if enabled
// Always returns a new instance (copy+merge)
func (f *AdvancedFormatter) MergeDetailsToFields(entry *log.Entry) log.Fields {
	if (f.Flags & FlagExtractDetails) > 0 {
		if err := GetError(entry); err != nil {
			// entry.With* does not copy Level, Caller, Message, Buffer
			return entry.WithFields(log.Fields(keyval.ToMap(errors.GetDetails(err)))).Data
		}
	}

	data := log.Fields{}
	for k, v := range entry.Data {
		data[k] = v
	}

	return data
}

// AppendDetailsToEntry appends errors.Details to logrus.Entry.Data, if not set yet
// implements logrus.Hook.Fire()
func AppendDetailsToEntry(entry *log.Entry) error {
	if err := GetError(entry); err != nil {
		for key, val := range keyval.ToMap(errors.GetDetails(err)) {
			if _, has := entry.Data[key]; !has {
				entry.Data[key] = val
			}
		}
	}
	return nil
}

// GetCallStack extracts simplified call stack from errors.StackTracer, if enabled
func (f *AdvancedFormatter) GetCallStack(entry *log.Entry) []string {
	if (f.Flags & (FlagCallStackInFields | FlagCallStackOnConsole | FlagCallStackInHTTPProblem)) > 0 {
		callStackLines := GetCallStack(entry)
		if len(callStackLines) > f.CallStackSkipLastX {
			return callStackLines[:len(callStackLines)-f.CallStackSkipLastX]
		}
	}

	return []string{}
}

// GetCallStack extracts simplified call stack from errors.StackTracer
func GetCallStack(entry *log.Entry) []string {
	if err := GetError(entry); err != nil {
		var stackTracer StackTracer
		if errors.As(err, &stackTracer) {
			return buildCallStackLines(stackTracer)
		}
	}

	return []string{}
}

// AppendDetailsToEntry appends CallStack to logrus.Entry.Data
// implements logrus.Hook.Fire()
func AppendCallStackToEntry(callStackSkipLast int) func(entry *log.Entry) error {
	return func(entry *log.Entry) error {
		callStackLines := GetCallStack(entry)
		if len(callStackLines) > callStackSkipLast {
			entry.Data[KeyCallStack] = callStackLines[:len(callStackLines)-callStackSkipLast]
		}

		return nil
	}
}

/*RenderFieldValues renders Details with field values (%+v), if enabled
Forces rendering error by Error()
*/
func (f *AdvancedFormatter) RenderFieldValues(data log.Fields) {
	for key, value := range data {
		if val := reflect.ValueOf(value); val.IsValid() {
			err, isError := value.(error) // %+v prints out stack trace, too
			if isError && err != nil {
				data[key] = err.Error()
			} else if (f.Flags & FlagPrintStructFieldNames) > 0 {
				if val.Kind() != reflect.String && !IsNumeric(val.Kind()) {
					data[key] = fmt.Sprintf("%+v", value)
				}
			}
		}
	}
}

// RenderErrorInEntry calls Error() on error values in logrus.Entry.Data and sets it back
// implements logrus.Hook.Fire()
// does not print out stack trace of errors (if %+v is used)
// JSON marshaller won't skip the error value
// WARNING: error values in logrus.Entry.Data cannot be dig anymore! (see: Format())
func RenderErrorInEntry(entry *log.Entry) error {
	for key, value := range entry.Data {
		if val := reflect.ValueOf(value); val.IsValid() {
			err, isError := value.(error) // %+v prints out stack trace, too
			if isError && err != nil {
				entry.Data[key] = err.Error()
			}
		}
	}

	return nil
}

// RenderStructFieldNames calls Sprintf("%+v", ...) on values in logrus.Entry.Data and sets it back
// implements logrus.Hook.Fire()
func RenderStructFieldNames(entry *log.Entry) error {
	for key, value := range entry.Data {
		if val := reflect.ValueOf(value); val.IsValid() {
			_, isError := value.(error)
			if !isError && val.Kind() != reflect.String && !IsNumeric(val.Kind()) {
				entry.Data[key] = fmt.Sprintf("%+v", value)
			}
		}
	}

	return nil
}

// AppendCallStack appends call stack (for the console print), if enabled
func (f *AdvancedFormatter) AppendCallStack(textPart []byte, callStackLines []string) []byte {
	if (f.Flags&FlagCallStackOnConsole) > 0 && len(callStackLines) > 0 {
		if len(textPart) > 0 && textPart[len(textPart)-1] != '\n' {
			textPart = append(textPart, '\n')
		}

		textPart = append(textPart, '\t')
		textPart = append(textPart,
			[]byte(strings.Join(callStackLines, "\n\t"))...,
		)
		textPart = append(textPart, '\n')
	}

	return textPart
}

func AppendCallStack(textPart []byte, callStackLines []string) []byte {
	if len(textPart) > 0 && textPart[len(textPart)-1] != '\n' {
		textPart = append(textPart, '\n')
	}

	textPart = append(textPart, '\t')
	textPart = append(textPart,
		[]byte(strings.Join(callStackLines, "\n\t"))...,
	)
	textPart = append(textPart, '\n')

	return textPart
}
