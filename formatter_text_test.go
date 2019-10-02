package errorformatter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

// nolint:unparam
func newTextLoggerMock(callStackSkipLast int, callStackNewLines bool, printStructFieldNames bool) *LoggerMock {
	RegisterSkipPackageFromStackTrace(pkgPathMarker{})

	logger := NewTextLogger(log.InfoLevel, callStackSkipLast, callStackNewLines, printStructFieldNames)
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

func TestLogrus_WithErrorDetailsCallStack_CallStackNewLines(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithErrorDetailsCallStack(err).Log(log.ErrorLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+StaticTimeFormat+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
	errorformatter.newWithDetails() errorformatter_test.go:0
	errorformatter.makeDeepErrors() errorformatter_test.go:0
	`+funcName+`() formatter_text_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithErrorDetailsCallStack_PrintStructFieldNames(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, true)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithErrorDetailsCallStack(err).Log(log.ErrorLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+StaticTimeFormat+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{Text:text Integer:42 Bool:true hidden:hidden}"
	errorformatter.newWithDetails() errorformatter_test.go:0
	errorformatter.makeDeepErrors() errorformatter_test.go:0
	`+funcName+`() formatter_text_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}
func TestErrors_WithErrorDetailsCallStack_CallStackInFields(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, false, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithErrorDetailsCallStack(err).Log(log.ErrorLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+StaticTimeFormat+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}" callstack="[errorformatter.newWithDetails() errorformatter_test.go:0 errorformatter.makeDeepErrors() errorformatter_test.go:0 `+funcName+`() formatter_text_test.go:0]"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_TextLogger_Info(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.Log(log.InfoLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=info time="`+StaticTimeFormat+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_TextLogger_WithError_Info(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithError(err).Info("WithError")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=info time="`+StaticTimeFormat+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg=WithError file="formatter_text_test.go:0"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithErrorDetails(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithErrorDetails(log.ErrorLevel, err).Log(log.ErrorLevel, err)

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=error time="`+StaticTimeFormat+`" func=`+funcName+` msg="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestLogrus_WithErrorDetails_WithError(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(2, true, false)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedTextFormatter)
	assert.True(t, ok, "AdvancedTextFormatter")
	formatter.TimestampFormat = StaticTimeFormat

	err := makeDeepErrors()
	loggerMock.WithErrorDetails(log.ErrorLevel, err).WithError(err).Warning("Warning")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `level=warning time="`+StaticTimeFormat+`" func=`+funcName+` error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg=Warning file="formatter_text_test.go:0" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
`, replaceCallLine(loggerMock.outBuf.String()))
}
