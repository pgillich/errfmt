package errorformatter

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogrus_RenderHTTPProblem_CallStackNewLines(t *testing.T) {
	funcName := FunctionNameShort()
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackInHTTPProblem,
		2)
	ts := time.Now()
	tsRFC3339 := ts.Format(time.RFC3339)

	err := GenerateDeepErrors()
	respBody, problemErr := RenderHTTPProblem(
		loggerMock.WithError(err).WithTime(ts), log.ErrorLevel, http.StatusPreconditionFailed,
	) /*.Log(log.ErrorLevel, "USER MSG")*/
	assert.Nil(t, problemErr, fmt.Sprintf("%s", problemErr))
	respText := string(respBody)

	if debugTest {
		fmt.Printf("###\n%s\n###\n%s\n###\n", loggerMock.outBuf.String(), respText)
	}
	// nolint:lll
	assert.Equal(t, `{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "K0_1": "\"V0_1\"",
    "K0_2": "\"V0_2\"",
    "K1_1": "\"V1_1\"",
    "K1_2": "\"V1_2\"",
    "K3 2": "\"V3 space\"",
    "K3\"5": "\"V3\\\"doublequote\"",
    "K3%6": "\"V3%percent\"",
    "K3:3": "\"V3:column\"",
    "K3;3": "\"V3;semicolumn\"",
    "K3=1": "\"V3=equal\"",
    "K5_bool": "true",
    "K5_int": "12",
    "K5_map": "{\"1\":\"ONE\",\"2\":\"TWO\"}",
    "K5_struct": "{\"Text\":\"text\",\"Integer\":42,\"Bool\":true}",
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"`+tsRFC3339+`\""
  },
  "callstack": [
    "errorformatter.newWithDetails() errorformatter_test.go:0",
    "errorformatter.GenerateDeepErrors() errorformatter_test.go:0",
    "`+funcName+`() formatter_html_test.go:0"
  ]
}`, replaceCallLine(respText))
}
