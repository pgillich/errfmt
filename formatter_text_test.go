package errorformatter

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

// nolint:unparam,lll
func newTextLoggerMock(flags int, callStackSkipLast int) *LoggerMock {
	RegisterSkipPackageFromStackTrace(pkgPathMarker{})

	logger := NewTextLogger(log.InfoLevel, flags, callStackSkipLast)
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

func TestLogrus_WithError_CallStackNewLines(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
	errorformatter.newWithDetails() errorformatter.go:0
	errorformatter.GenerateDeepErrors() errorformatter.go:0
	`+funcName+`() formatter_text_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithError_ExtractDetails_CallStackOnConsole(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
	errorformatter.newWithDetails() errorformatter.go:0
	errorformatter.GenerateDeepErrors() errorformatter.go:0
	`+funcName+`() formatter_text_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithError_ExtractDetails_CallStackOnConsole_PrintStructFieldNames(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole|FlagPrintStructFieldNames,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{Text:text Integer:42 Bool:true hidden:hidden}"
	errorformatter.newWithDetails() errorformatter.go:0
	errorformatter.GenerateDeepErrors() errorformatter.go:0
	`+funcName+`() formatter_text_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestErrors_WithError_ExtractDetails_CallStackInFields(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackInFields,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}" callstack="[errorformatter.newWithDetails() errorformatter.go:0 errorformatter.GenerateDeepErrors() errorformatter.go:0 `+funcName+`() formatter_text_test.go:0]"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_TextLogger_Info(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)
	/*
		formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
		assert.True(t, ok, "AdvancedTextFormatter")
		formatter.TimestampFormat = StaticTimeFormat
	*/

	err := GenerateDeepErrors()
	loggerMock.WithTime(ts).Log(log.InfoLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=info time="`+tsRFC3339+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_TextLogger_WithError_Info(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagNone,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Info("USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=info time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithError_ExtractDetails(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+tsRFC3339+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
`, replaceCallLine(loggerMock.outBuf.String()))
}
