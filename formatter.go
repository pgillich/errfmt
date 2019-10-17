package errorformatter

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
	CallStackSkipLast int
}

// GetError extracts error from entry.Data (a result of log.WithError())
func (f *AdvancedFormatter) GetError(entry *log.Entry) error {
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
	if (f.Flags & FlagCallStackInFields) > 0 {
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

// MergeDetailsToFields merges Details from error to, if enabled
// Always returns a new instance (copy+merge)
func (f *AdvancedFormatter) MergeDetailsToFields(entry *log.Entry) log.Fields {
	if (f.Flags & FlagExtractDetails) > 0 {
		if err := f.GetError(entry); err != nil {
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

// GetCallStack extracts simplified call stack from errors.StackTracer, if enabled
func (f *AdvancedFormatter) GetCallStack(entry *log.Entry) []string {
	if (f.Flags & (FlagCallStackInFields | FlagCallStackOnConsole | FlagCallStackInHTTPProblem)) > 0 {
		if err := f.GetError(entry); err != nil {
			var stackTracer StackTracer
			if errors.As(err, &stackTracer) {
				callStackLines := buildCallStackLines(stackTracer)
				if len(callStackLines) > f.CallStackSkipLast {
					return callStackLines[:len(callStackLines)-f.CallStackSkipLast]
				}
			}
		}
	}

	return []string{}
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

// GetClashingFields returns the automatical filles fields
func (f *AdvancedFormatter) GetClashingFields() []string {
	return []string{
		log.FieldKeyLevel, log.FieldKeyTime, log.FieldKeyFunc,
		log.FieldKeyMsg, log.FieldKeyFile, KeyCallStack,
	}
}
