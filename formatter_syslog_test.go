package errorformatter

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/juju/rfc/rfc5424"
	log "github.com/sirupsen/logrus"
)

// nolint:unparam
func newSyslogLoggerMock(flags int, callStackSkipLast int) *LoggerMock {
	RegisterSkipPackageFromStackTrace(pkgPathMarker{})

	logger := NewSyslogLogger(log.InfoLevel,
		flags, callStackSkipLast,
		rfc5424.FacilityDaemon, rfc5424.Hostname{FQDN: "fqdn.host.com"}, "application",
		"PID", "",
	)
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

func TestSyslog_WithErrorDetailsCallStack_CallStackNewLines(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newSyslogLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole|FlagTrimJSONDquote,
		2)
	formatter, ok := loggerMock.Logger.Formatter.(*AdvancedSyslogFormatter)
	assert.True(t, ok, "AdvancedSyslogFormatter")
	assert.NotNil(t, formatter, "AdvancedSyslogFormatter")
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339Nano)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `<27>1 `+tsRFC3339+` fqdn.host.com application PID DETAILS_MSG [details level="error" func="`+funcName+`" error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax" file="formatter_syslog_test.go:0" K0_1="V0_1" K0_2="V0_2" K1_1="V1_1" K1_2="V1_2" K3_2="V3 space" K3_5="V3\\\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3_1="V3=equal" K5_bool="true" K5_int="12" K5_map="{\"1\":\"ONE\",\"2\":\"TWO\"}" K5_struct="{\"Text\":\"text\",\"Integer\":42,\"Bool\":true}"] USER MSG
	errorformatter.newWithDetails() errorformatter.go:0
	errorformatter.GenerateDeepErrors() errorformatter.go:0
	`+funcName+`() formatter_syslog_test.go:0
`, replaceCallLine(loggerMock.outBuf.String()))
}

func TestSyslog_WithErrorDetailsCallStack_CallStackInFields(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newSyslogLoggerMock(
		FlagExtractDetails|FlagCallStackInFields|FlagTrimJSONDquote,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339Nano)

	err := GenerateDeepErrors()
	loggerMock.WithError(err).WithTime(ts).Log(log.ErrorLevel, "USER MSG")

	if debugTest {
		fmt.Printf("###\n%s\n###\n", loggerMock.outBuf.String())
	}
	// nolint:lll
	assert.Equal(t, `<27>1 `+tsRFC3339+` fqdn.host.com application PID DETAILS_MSG [details level="error" func="`+funcName+`" error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax" file="formatter_syslog_test.go:0" K0_1="V0_1" K0_2="V0_2" K1_1="V1_1" K1_2="V1_2" K3_2="V3 space" K3_5="V3\\\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3_1="V3=equal" K5_bool="true" K5_int="12" K5_map="{\"1\":\"ONE\",\"2\":\"TWO\"}" K5_struct="{\"Text\":\"text\",\"Integer\":42,\"Bool\":true}" callstack="[\"errorformatter.newWithDetails() errorformatter.go:0\",\"errorformatter.GenerateDeepErrors() errorformatter.go:0\",\"`+funcName+`() formatter_syslog_test.go:0\"\]"] USER MSG`, replaceCallLine(loggerMock.outBuf.String()))
}
