/*
	Package errorformatter is a Golang library for formatting logrus + emperror/errors messages
*/
package errorformatter

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"emperror.dev/errors"
	"emperror.dev/errors/utils/keyval"
	log "github.com/sirupsen/logrus"
)

const (
	// KeyCallStack is a context key and field name for call stack
	KeyCallStack = "callstack"
	// MaximumCallerDepth is the max. call stack deep
	MaximumCallerDepth = 50
)

var (
	skipPackageNameForCaller = make(map[string]struct{}, 1) // nolint:gochecknoglobals
)

// AddSkipPackageFromStackTrace adds package name for trimming
func AddSkipPackageFromStackTrace(name string) {
	skipPackageNameForCaller[name] = struct{}{}
}

// RegisterSkipPackageFromStackTrace registers package name of given variable (from main) for trimming
func RegisterSkipPackageFromStackTrace(v interface{}) {
	pkgPath := reflect.TypeOf(v).PkgPath()
	if slash := strings.LastIndex(pkgPath, "/"); slash >= 0 {
		AddSkipPackageFromStackTrace(pkgPath[:slash])
	}
}

// StackTracer is an interface type to find stack trace of errors chain
type StackTracer interface {
	StackTrace() errors.StackTrace
}

// ContextLogFieldKey is the type of field keys in log context
type ContextLogFieldKey string

// AdvancedLogger is a decorator struct to Logrus Logger
type AdvancedLogger struct {
	*log.Logger
	/* CallStackSkipLast at ErrorWithCallStack()
	If 0, call stack lines are NOT printed (=disabled)
	If >0, call stack lines are printed, skipping the last lines
		so, the main() will never be printed.
	*/
	CallStackSkipLast int
	/* CallStackNewLines at ErrorWithCallStack()
	true: call stack is printed in new lines
	false: call stack is appended to Details with field "callstack"
	*/
	CallStackNewLines bool
	//PrintStructFieldNames bool
}

// ErrorWithCallStack prints out call stack, too
func (logger *AdvancedLogger) ErrorWithCallStack(level log.Level, err error) {
	var entry *log.Entry
	fields := keyval.ToMap(errors.GetDetails(err))

	var stackTracer StackTracer
	if logger.CallStackSkipLast > 0 && errors.As(err, &stackTracer) {
		callStackLines := buildCallStackLines(stackTracer)
		if len(callStackLines) > logger.CallStackSkipLast {
			callStackLines = callStackLines[:len(callStackLines)-logger.CallStackSkipLast]
			if logger.CallStackNewLines {
				ctxCallStack := context.WithValue(context.Background(),
					ContextLogFieldKey(KeyCallStack), callStackLines,
				)
				entry = logger.WithContext(ctxCallStack).WithFields(log.Fields(fields))
				entry.Log(level, err)
				return
			}
			fields[KeyCallStack] = callStackLines
		}
	}

	entry = logger.WithFields(log.Fields(fields))
	entry.Log(level, err)
}

/*
ModuleCallerPrettyfier trims registered package name(s)
	Fits to TextFormatter.CallerPrettyfier
	Similar pull request: https://github.com/sirupsen/logrus/pull/989
*/
func ModuleCallerPrettyfier(frame *runtime.Frame) (string, string) {
	filePath := frame.File
	if i := strings.LastIndex(filePath, "/"); i >= 0 {
		filePath = filePath[i+1:]
	}

	return TrimModuleNamePrefix(frame.Function), fmt.Sprintf("%s:%d", filePath, frame.Line)
}

// TrimModuleNamePrefix trims package name(s)
func TrimModuleNamePrefix(functionName string) string {
	for prefix := range skipPackageNameForCaller {
		prefixDot := prefix + "."
		prefixSlash := prefix + "/"
		if strings.HasPrefix(functionName, prefixDot) {
			functionName = strings.TrimPrefix(functionName, prefixDot)
			break
		} else if strings.HasPrefix(functionName, prefixSlash) {
			functionName = strings.TrimPrefix(functionName, prefixSlash)
			break
		}
	}

	return functionName
}

/*
ParentCallerHook is a Logrus Hook implementation
	Patches the "func" and "file" fields to the parent
	Similar pull request: https://github.com/sirupsen/logrus/pull/973
	Unfortunately, sirupsen/logrus/Entry.log() always overwrites Entry.Caller,
		instead of leaving the patched value, if it's not nil
*/
type ParentCallerHook struct {
	ParentCount int
}

// Levels returns all levels
func (*ParentCallerHook) Levels() []log.Level {
	return log.AllLevels
}

// Fire replaces Entry.Caller to the parent
func (h *ParentCallerHook) Fire(entry *log.Entry) error {
	if h.ParentCount <= 0 || entry.Caller == nil {
		return nil
	}

	pcs := make([]uintptr, MaximumCallerDepth)
	depth := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	var parentCount = MaximumCallerDepth - 1
	var f runtime.Frame
	var again bool
	for f, again = frames.Next(); again && parentCount > 0; f, again = frames.Next() {
		if f == *entry.Caller {
			parentCount = h.ParentCount
		}
		parentCount--
	}
	if again { // for loop exited by parentCount == 0
		entry.Caller = &f // nolint:scopelint
	}

	return nil
}

// buildCallStackLines builds a compact list of call stack lines
func buildCallStackLines(stackTracer StackTracer) []string {
	callStackLines := []string{}

	stackTrace := stackTracer.StackTrace()
	for _, t := range stackTrace {
		dsFunction := dummyState{flags: map[int]bool{'+': true}}
		t.Format(&dsFunction, 's')
		functionName := TrimModuleNamePrefix(strings.Split(dsFunction.str.String(), "\n")[0])

		dsPath := dummyState{}
		t.Format(&dsPath, 's')
		path := dsPath.str.String()

		dsLine := dummyState{}
		t.Format(&dsLine, 'd')
		line := dsLine.str.String()

		callStackLines = append(callStackLines, fmt.Sprintf("%s() %s:%s", functionName, path, line))
	}

	return callStackLines
}

/* dummyState is a dummy, in-memory fmt.State implementation
Used for reading StackTracer items
*/
type dummyState struct {
	str   strings.Builder
	flags map[int]bool
}

// Write pass trough strings.Builder
func (ds *dummyState) Write(b []byte) (n int, err error) {
	return ds.str.Write(b)
}

// Width not implemented
func (*dummyState) Width() (wid int, ok bool) {
	return 0, false
}

// Precision not implemented
func (*dummyState) Precision() (prec int, ok bool) {
	return 0, false
}

// Flag returns dummyState.flags
func (ds *dummyState) Flag(c int) bool {
	if f, ok := ds.flags[c]; ok {
		return f
	}

	return false
}

// FunctionName returns the actual function name (long)
func FunctionName() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

// FunctionNameShort returns the actual short function name (w/o package)
func FunctionNameShort() string {
	pc, _, _, _ := runtime.Caller(1)
	longName := runtime.FuncForPC(pc).Name()
	return longName[strings.LastIndex(longName, "/")+1:]
}

// CallerFunctionName returns the calles (long) function name
func CallerFunctionName() string {
	pc, _, _, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name()
}

// IsNumeric returns true, if it's a number kind
func IsNumeric(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.Complex64, reflect.Complex128:
		return true
	}

	return false
}
