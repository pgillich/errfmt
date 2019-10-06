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

// FillDetailsToFields extracts Details from error, if enabled
func (f *AdvancedFormatter) FillDetailsToFields(entry *log.Entry) {
	if (f.Flags & FlagExtractDetails) > 0 {
		if err := f.GetError(entry); err != nil {
			// entry.With* does not copy Level, Caller, Message, Buffer
			entry.Data = entry.WithFields(log.Fields(keyval.ToMap(errors.GetDetails(err)))).Data
		}
	}
}

/*FillCallStack extracts simplified call stack from errors.StackTracer, if enabled
if FlagCallStackInFields, puts into entry.Data
if FlagCallStackOnConsole, returns the call stack for printing it later on
if FlagCallStackInHTTPProblem, ???
*/
func (f *AdvancedFormatter) FillCallStack(entry *log.Entry) []string {
	if (f.Flags & (FlagCallStackInFields | FlagCallStackOnConsole | FlagCallStackInHTTPProblem)) > 0 {
		if err := f.GetError(entry); err != nil {
			var stackTracer StackTracer
			if errors.As(err, &stackTracer) {
				callStackLines := buildCallStackLines(stackTracer)
				if len(callStackLines) > f.CallStackSkipLast {
					callStackLines = callStackLines[:len(callStackLines)-f.CallStackSkipLast]
					if (f.Flags & FlagCallStackInFields) > 0 {
						entry.Data[KeyCallStack] = callStackLines
					}
					if (f.Flags & FlagCallStackOnConsole) > 0 {
						return callStackLines
					}
				}
			}
		}
	}

	return []string{}
}

/*RenderFieldValues renders Details with field values (%+v), if enabled
Forces rendering error by Error()
*/
func (f *AdvancedFormatter) RenderFieldValues(entry *log.Entry) {
	for key, value := range entry.Data {
		if val := reflect.ValueOf(value); val.IsValid() {
			err, isError := value.(error) // %+v prints out stack trace, too
			if isError && err != nil {
				entry.Data[key] = err.Error()
			} else if (f.Flags & FlagPrintStructFieldNames) > 0 {
				if val.Kind() != reflect.String && !IsNumeric(val.Kind()) {
					entry.Data[key] = fmt.Sprintf("%+v", value)
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
