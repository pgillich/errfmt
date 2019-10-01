package errorformatter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

func newJSONLoggerMock(callStackSkipLast int, callStackNewLines bool) *LoggerMock {
	RegisterSkipPackageFromStackTrace(pkgPathMarker{})

	logger := NewJSONLogger(log.InfoLevel, callStackSkipLast, callStackNewLines)
	buf := new(bytes.Buffer)
	loggerMock := &LoggerMock{
		AdvancedLogger: logger,
		outBuf:         buf,
		exitCode:       -1,
	}
	loggerMock.Out = buf
	loggerMock.ExitFunc = loggerMock.exit

	return loggerMock
}

func TestLogrus_JSONLogger(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newJSONLoggerMock(2, true)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedJSONFormatter)
	assert.True(t, ok, "AdvancedJSONFormatter")
	formatter.TimestampFormat = StaticTimeFormat
	formatter.PrettyPrint = true

	err := makeDeepErrors()

	loggerMock.ErrorWithCallStack(log.ErrorLevel, err)
	fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	// nolint:lll
	assert.Equal(t, `{
  "K0_1": "V0_1",
  "K0_2": "V0_2",
  "K1_1": "V1_1",
  "K1_2": "V1_2",
  "K3 2": "V3 space",
  "K3\"5": "V3\"doublequote",
  "K3%6": "V3%percent",
  "K3:3": "V3:column",
  "K3;3": "V3;semicolumn",
  "K3=1": "V3=equal",
  "K5_bool": true,
  "K5_int": 12,
  "K5_map": {
    "1": "ONE",
    "2": "TWO"
  },
  "K5_struct": {
    "Text": "text",
    "Integer": 42,
    "Bool": true
  },
  "file": "formatter_json_test.go:0",
  "func": "`+funcName+`",
  "level": "error",
  "msg": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "time": "`+StaticTimeFormat+`"
}
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_JSONLogger_CallStackInFields(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newJSONLoggerMock(2, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedJSONFormatter)
	assert.True(t, ok, "AdvancedJSONFormatter")
	formatter.TimestampFormat = StaticTimeFormat
	formatter.PrettyPrint = true

	err := makeDeepErrors()

	loggerMock.ErrorWithCallStack(log.ErrorLevel, err)
	fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	// nolint:lll
	assert.Equal(t, `{
  "K0_1": "V0_1",
  "K0_2": "V0_2",
  "K1_1": "V1_1",
  "K1_2": "V1_2",
  "K3 2": "V3 space",
  "K3\"5": "V3\"doublequote",
  "K3%6": "V3%percent",
  "K3:3": "V3:column",
  "K3;3": "V3;semicolumn",
  "K3=1": "V3=equal",
  "K5_bool": true,
  "K5_int": 12,
  "K5_map": {
    "1": "ONE",
    "2": "TWO"
  },
  "K5_struct": {
    "Text": "text",
    "Integer": 42,
    "Bool": true
  },
  "callstack": [
    "errorformatter.newWithDetails() errorformatter_test.go:0",
    "errorformatter.makeDeepErrors() errorformatter_test.go:0",
    "`+funcName+`() formatter_json_test.go:0"
  ],
  "file": "formatter_json_test.go:0",
  "func": "`+funcName+`",
  "level": "error",
  "msg": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "time": "`+StaticTimeFormat+`"
}
`, replaceCallLine(loggerMock.outBuf.String()))
}
