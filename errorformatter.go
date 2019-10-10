/*
	Package errorformatter is a Golang library for formatting logrus + emperror/errors messages
*/
package errorformatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// KeyCallStack is a context key and field name for call stack
	KeyCallStack = "callstack"
	// MaximumCallerDepth is the max. call stack deep
	MaximumCallerDepth = 50
	// DisabledFieldWeight is the value for dropping the field during ordering
	DisabledFieldWeight = -100

	// FlagNone disables all flags of AdvancedLogger.Flags
	FlagNone = 0
	// FlagExtractDetails extracts errors.Details to logrus.Fields
	FlagExtractDetails = 1 << 0
	// FlagCallStackInFields extracts errors.StackTrace() to logrus.Fields
	FlagCallStackInFields = 1 << 1
	// FlagCallStackOnConsole extracts errors.StackTrace() to logrus.Field "callstack"
	FlagCallStackOnConsole = 1 << 2
	// FlagCallStackInHTTPProblem extracts errors.StackTrace() to HTTPProblem
	FlagCallStackInHTTPProblem = 1 << 3
	// FlagPrintStructFieldNames renders non-scalar Details values are by "%+v", instead of "%v"
	FlagPrintStructFieldNames = 1 << 4
	// FlagTrimJSONDquote trims the leading and trailing '"' of JSON-formatted values
	FlagTrimJSONDquote = 1 << 5
)

var (
	skipPackageNameForCaller = make(map[string]struct{}, 1) // nolint:gochecknoglobals
	debugTest                = true                         // nolint:gochecknoglobals
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
	pc, _, _, _ := runtime.Caller(1) // nolint:dogsled
	return runtime.FuncForPC(pc).Name()
}

// FunctionNameShort returns the actual short function name (w/o package)
func FunctionNameShort() string {
	pc, _, _, _ := runtime.Caller(1) // nolint:dogsled
	longName := runtime.FuncForPC(pc).Name()
	return longName[strings.LastIndex(longName, "/")+1:]
}

// CallerFunctionName returns the calles (long) function name
func CallerFunctionName() string {
	pc, _, _, _ := runtime.Caller(2) // nolint:dogsled
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

/*
JSONMarshal marshals with escapeHTML flag.
	if indent is "", indenting is disabled.
	Removes last endline.
*/
func JSONMarshal(t interface{}, indent string, escapeHTML bool) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	if len(indent) > 0 {
		encoder.SetIndent("", indent)
	}
	encoder.SetEscapeHTML(escapeHTML)
	err := encoder.Encode(t)
	jsonBytes := buffer.Bytes()
	if len(jsonBytes) > 0 && jsonBytes[len(jsonBytes)-1] == '\n' {
		jsonBytes = jsonBytes[0 : len(jsonBytes)-1]
	}
	return jsonBytes, err
}

// nolint:golint
func SetEntryTimestamp(entry *log.Entry, ts time.Time) *log.Entry {
	entry.Time = ts
	return entry
}

// nolint:golint
func DigErrorsString(err error) string {
	if msg := GetFieldString(err, "msg"); msg != nil {
		return *msg
	}

	return err.Error()
}

// nolint:golint
func GetFieldString(input interface{}, keyName string) *string {
	if input == nil {
		return nil
	}
	var inputVal reflect.Value
	if inputVal = reflect.ValueOf(input); !inputVal.IsValid() {
		return nil
	}

	if inputVal.Kind() == reflect.Ptr {
		if inputVal.IsNil() {
			return nil
		}
		inputVal = reflect.Indirect(inputVal)
		if !inputVal.IsValid() {
			return nil
		}
	}
	if inputVal.Kind() == reflect.Struct {
		typ := inputVal.Type()
		for i := 0; i < typ.NumField(); i++ {
			if typ.Field(i).Name == keyName {
				if field := inputVal.Field(i); field.IsValid() {
					fieldValue := fmt.Sprintf("%+v", field)
					return &fieldValue
				}
				break
			}
		}
	}

	return nil
}

// GenerateDeepErrors generates test error
func GenerateDeepErrors() error {
	type complexStruct struct {
		Text    string
		Integer int
		Bool    bool
		hidden  string
	}

	err := newWithDetails()
	err = errors.WithDetails(err, "K1_1", "V1_1", "K1_2", "V1_2")
	err = errors.WithMessage(err, "MESSAGE:2")
	err = errors.WithDetails(err,
		"K3=1", "V3=equal",
		"K3 2", "V3 space",
		"K3;3", "V3;semicolumn",
		"K3:3", "V3:column",
		`K3"5`, `V3"doublequote`,
		"K3%6", "V3%percent",
	)
	err = errors.WithMessage(err, "MESSAGE 4")
	err = errors.WithDetails(err,
		"K5_int", 12,
		"K5_bool", true,
		"K5_struct", complexStruct{Text: "text", Integer: 42, Bool: true, hidden: "hidden"},
		"K5_map", map[int]string{1: "ONE", 2: "TWO"},
	)

	return err
}

func newWithDetails() error {
	_, err := strconv.Atoi("NO_NUMBER")
	return errors.WrapWithDetails(err, "MESSAGE%0", "K0_1", "V0_1", "K0_2", "V0_2")
}
