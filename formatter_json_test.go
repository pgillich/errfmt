package errorformatter

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

func newJSONLoggerMock(flags int, callStackSkipLast int) *LoggerMock {
	RegisterSkipPackageFromStackTrace(pkgPathMarker{})

	logger := NewJSONLogger(log.InfoLevel, flags, callStackSkipLast)
	buf := new(bytes.Buffer)
	loggerMock := &LoggerMock{
		Logger:   logger,
		outBuf:   buf,
		exitCode: -1,
	}
	loggerMock.Out = buf
	loggerMock.ExitFunc = loggerMock.exit

	return loggerMock
}

func TestLogrus_JSONLogger(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newJSONLoggerMock(
		FlagExtractDetails,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedJSONFormatter)
	assert.True(t, ok, "AdvancedJSONFormatter")
	formatter.PrettyPrint = true

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
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
  "error": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "file": "formatter_json_test.go:0",
  "func": "`+funcName+`",
  "level": "error",
  "msg": "USER MSG",
  "time": "`+tsRFC3339+`"
}
`, replaceCallLine(loggerMock.outBuf.String()))
	}
}

func TestLogrus_JSONLogger_CallStackInFields(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newJSONLoggerMock(
		FlagExtractDetails|FlagCallStackInFields,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedJSONFormatter)
	assert.True(t, ok, "AdvancedJSONFormatter")
	formatter.PrettyPrint = true

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
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
    "errorformatter.newWithDetails() errorformatter.go:0",
    "errorformatter.GenerateDeepErrors() errorformatter.go:0",
    "`+funcName+`() formatter_json_test.go:0"
  ],
  "error": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "file": "formatter_json_test.go:0",
  "func": "`+funcName+`",
  "level": "error",
  "msg": "USER MSG",
  "time": "`+tsRFC3339+`"
}
`, replaceCallLine(loggerMock.outBuf.String()))
	}
}
