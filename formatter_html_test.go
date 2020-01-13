package errfmt

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"emperror.dev/errors"
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
	respBody, problemErr := RenderHTTPProblem(http.StatusPreconditionFailed,
		loggerMock.WithError(err).WithTime(ts),
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
    "time": "\"`+tsRFC3339+`\""
  },
  "callstack": [
    "errfmt.newWithDetails() errfmt.go:0",
    "errfmt.GenerateDeepErrors() errfmt.go:0",
    "`+funcName+`() formatter_html_test.go:0"
  ]
}`, replaceCallLine(respText))
}

// implements HTTPHandlerWithLoggerFunc
func handleTest(w http.ResponseWriter, r *http.Request, logger *log.Logger) {
	AddHookRequestInfo(logger, r, DefaultSelectedRequestInfo())

	logger.WithField("FIELD", "VALUE").Info("Message")

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Hello")) // nolint:errcheck,gosec
}

func TestEmperror_HTTP(t *testing.T) {
	handleFunc := handleTest
	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole,
		1)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/test", HTTPHandlerWithLogger(handleFunc, loggerMock.Logger))

	testServer := httptest.NewServer(serveMux)
	defer testServer.Close()

	client := http.Client{}
	reqURL := (&url.URL{
		Scheme:   "http",
		Host:     testServer.Listener.Addr().String(),
		Path:     "test",
		RawQuery: "key=val",
	}).String()

	req, err := http.NewRequest("GET", reqURL, nil)
	assert.Nil(t, err, fmt.Sprintf("%s", err))

	req.Header.Add("From", "user@example.com")
	req.Header.Add("Forwarded", "for=192.0.2.60;proto=http;by=203.0.113.43")
	req.Header.Add("Content-Length", "12")
	req.Header.Add("X-Forwarded-For", "client1, proxy1, proxy2")
	req.Header.Add("X-Forwarded-Host", "en.wikipedia.org:8080")
	req.Header.Add("X-HTTP-Method-Override", "DELETE")

	resp, err := client.Do(req)
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, resp.StatusCode, http.StatusAccepted, "Status")

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close() //nolint:gosec,errcheck
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, "Hello", string(body), "Body")

	handleFuncName := runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()
	handleFuncShortName := handleFuncName[strings.LastIndex(handleFuncName, "/")+1:]

	logLine := loggerMock.outBuf.String()
	patternItems := []string{
		`^level=info`,
		`time="[0-9-T:+]+"`,
		`func=` + handleFuncShortName,
		`msg=Message`,
		`file="formatter_html_test\.go:[0-9]+"`,
		`FIELD=VALUE`,
		`req_Forwarded="for=192\.0\.2\.60;proto=http;by=203\.0\.113\.43"`,
		`req_From=user@example\.com`,
		`req_X-Forwarded-For="client1, proxy1, proxy2"`,
		`req_X-Forwarded-Host="en\.wikipedia\.org:8080"`,
		`req_X-Http-Method-Override=DELETE`,
		`req_host="127\.0\.0\.1:[0-9]+"`,
		`req_method=GET`,
		`req_remoteaddr="127\.0\.0\.1:[0-9]+"`,
		`req_requesturi="\/test\?key=val"
$`,
	}

	assert.Regexp(t,
		strings.Join(patternItems, " "),
		logLine,
		"Log",
	)
}

//handleTestAccepted implements HTTPHandlerWithErrorFunc
// nolint:unparam,deadcode,unused,nolint
func handleTestAccepted(w http.ResponseWriter, r *http.Request, logger *log.Logger,
) (jsonObj interface{}, status int, err error) {
	AddHookRequestInfo(logger, r, DefaultSelectedRequestInfo())

	logger.WithField("FIELD", "VALUE").Info("Message")

	response := struct{ Result string }{"OK"}

	return &response, http.StatusAccepted, err
}

func TestEmperror_HTTP_Accepted(t *testing.T) {
	handleFunc := handleTestAccepted

	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole,
		1)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/test",
		HTTPHandlerWithError(handleFunc, loggerMock.Logger, DefaultLevelByStatus()))

	testServer := httptest.NewServer(serveMux)
	defer testServer.Close()

	client := http.Client{}
	reqURL := (&url.URL{
		Scheme:   "http",
		Host:     testServer.Listener.Addr().String(),
		Path:     "test",
		RawQuery: "key=val",
	}).String()

	req, err := http.NewRequest("GET", reqURL, nil)
	assert.Nil(t, err, fmt.Sprintf("%s", err))

	req.Header.Add("From", "user@example.com")
	req.Header.Add("Forwarded", "for=192.0.2.60;proto=http;by=203.0.113.43")
	req.Header.Add("Content-Length", "12")
	req.Header.Add("X-Forwarded-For", "client1, proxy1, proxy2")
	req.Header.Add("X-Forwarded-Host", "en.wikipedia.org:8080")
	req.Header.Add("X-HTTP-Method-Override", "DELETE")

	resp, err := client.Do(req)
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "Status")

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close() //nolint:gosec,errcheck
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, `{
  "Result": "OK"
}`, string(body), "Body")

	handleFuncName := runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()
	handleFuncShortName := handleFuncName[strings.LastIndex(handleFuncName, "/")+1:]

	logLine := loggerMock.outBuf.String()
	patternItems := []string{
		`^level=info`,
		`time="[0-9-T:+]+"`,
		`func=` + handleFuncShortName,
		`msg=Message`,
		`file="formatter_html_test\.go:[0-9]+"`,
		`FIELD=VALUE`,
		`req_Forwarded="for=192\.0\.2\.60;proto=http;by=203\.0\.113\.43"`,
		`req_From=user@example\.com`,
		`req_X-Forwarded-For="client1, proxy1, proxy2"`,
		`req_X-Forwarded-Host="en\.wikipedia\.org:8080"`,
		`req_X-Http-Method-Override=DELETE`,
		`req_host="127\.0\.0\.1:[0-9]+"`,
		`req_method=GET`,
		`req_remoteaddr="127\.0\.0\.1:[0-9]+"`,
		`req_requesturi="\/test\?key=val"
$`,
	}

	assert.Regexp(t,
		strings.Join(patternItems, " "),
		logLine,
		"Log",
	)
}

//handleTestNotAcceptable implements HTTPHandlerWithErrorFunc
// nolint:unparam,deadcode,unused,nolint
func handleTestNotAcceptable(w http.ResponseWriter, r *http.Request, logger *log.Logger,
) (jsonObj interface{}, status int, err error) {
	AddHookRequestInfo(logger, r, DefaultSelectedRequestInfo())

	logger.WithField("FIELD", "VALUE").Info("Message")

	err = GenerateDeepErrors()

	return nil, http.StatusNotAcceptable, errors.WithMessage(err, "No luck")
}

func TestEmperror_HTTP_NotAcceptable(t *testing.T) {
	handleFunc := handleTestNotAcceptable

	loggerMock := newTextLoggerMock(
		FlagExtractDetails|FlagCallStackOnConsole|FlagCallStackInHTTPProblem,
		1)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/test",
		HTTPHandlerWithError(handleFunc, loggerMock.Logger, DefaultLevelByStatus()))

	testServer := httptest.NewServer(serveMux)
	defer testServer.Close()

	client := http.Client{}
	reqURL := (&url.URL{
		Scheme:   "http",
		Host:     testServer.Listener.Addr().String(),
		Path:     "test",
		RawQuery: "key=val",
	}).String()

	req, err := http.NewRequest("GET", reqURL, nil)
	assert.Nil(t, err, fmt.Sprintf("%s", err))

	req.Header.Add("From", "user@example.com")
	req.Header.Add("Forwarded", "for=192.0.2.60;proto=http;by=203.0.113.43")
	req.Header.Add("Content-Length", "12")
	req.Header.Add("X-Forwarded-For", "client1, proxy1, proxy2")
	req.Header.Add("X-Forwarded-Host", "en.wikipedia.org:8080")
	req.Header.Add("X-HTTP-Method-Override", "DELETE")

	resp, err := client.Do(req)
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, resp.StatusCode, http.StatusNotAcceptable, "Status")

	handleFuncName := runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()
	handleFuncShortName := handleFuncName[strings.LastIndex(handleFuncName, "/")+1:]

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close() //nolint:gosec,errcheck
	assert.Nil(t, err, fmt.Sprintf("%s", err))
	assert.Equal(t, `{
  "type": "about:blank",
  "title": "Not Acceptable",
  "status": 406,
  "detail": "No luck: MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
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
    "error": "\"No luck: MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "handlerFunc": "\"`+handleFuncName+`\"",
    "time": "\"RFC3339\""
  },
  "callstack": [
    "errfmt.newWithDetails() errfmt.go:0",
    "errfmt.GenerateDeepErrors() errfmt.go:0",
    "`+handleFuncShortName+`() formatter_html_test.go:0",
    "errfmt.HTTPHandlerWithError.func1() formatter_html.go:0",
    "net/http.HandlerFunc.ServeHTTP() server.go:0",
    "net/http.(*ServeMux).ServeHTTP() server.go:0",
    "net/http.serverHandler.ServeHTTP() server.go:0",
    "net/http.(*conn).serve() server.go:0"
  ]
}`, replaceTimestamp(replaceCallLine(string(body))), "Body")
	//	fmt.Println(replaceTimestamp(replaceCallLine(string(body))))

	logLine := replaceTimestamp(replaceCallLine(loggerMock.outBuf.String()))
	//fmt.Printf("\nlogLine:\n%s\n", logLine)

	patternInfoItems := []string{
		`^level=info`,
		`time="RFC3339"`,
		`func=` + handleFuncShortName,
		`msg=Message`,
		`file="formatter_html_test\.go:[0-9]+"`,
		`FIELD=VALUE`,
		`req_Forwarded="for=192\.0\.2\.60;proto=http;by=203\.0\.113\.43"`,
		`req_From=user@example\.com`,
		`req_X-Forwarded-For="client1, proxy1, proxy2"`,
		`req_X-Forwarded-Host="en\.wikipedia\.org:8080"`,
		`req_X-Http-Method-Override=DELETE`,
		`req_host="127\.0\.0\.1:[0-9]+"`,
		`req_method=GET`,
		`req_remoteaddr="127\.0\.0\.1:[0-9]+"`,
		`req_requesturi="\/test\?key=val"
`,
	}

	patternWarningItems := []string{
		`level=warning`,
		`time="RFC3339"`,
		`handlerFunc=` + regexp.QuoteMeta(handleFuncName),
		`func=errfmt\.HTTPHandlerWithError\.func1`,
		regexp.QuoteMeta(`error="No luck: MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax"`),
		`file="formatter_html\.go:0"`,
		`K0_1=V0_1`,
		`K0_2=V0_2`,
		`K1_1=V1_1`,
		`K1_2=V1_2`,
		`K3 2="V3 space"`,
		regexp.QuoteMeta(`K3"5="V3\"doublequote"`),
		`K3%6="V3%percent"`,
		`K3:3="V3:column"`,
		`K3;3="V3;semicolumn"`,
		`K3=1="V3=equal"`,
		`K5_bool=true`,
		`K5_int=12`,
		regexp.QuoteMeta(`K5_map="map[1:ONE 2:TWO]"`),
		regexp.QuoteMeta(`K5_struct="{text 42 true hidden}"`),
		`req_Forwarded="for=192\.0\.2\.60;proto=http;by=203\.0\.113\.43"`,
		`req_From=user@example\.com`,
		`req_X-Forwarded-For="client1, proxy1, proxy2"`,
		`req_X-Forwarded-Host="en\.wikipedia\.org:8080"`,
		`req_X-Http-Method-Override=DELETE`,
		`req_host="127\.0\.0\.1:[0-9]+"`,
		`req_method=GET`,
		`req_remoteaddr="127\.0\.0\.1:[0-9]+"`,
		regexp.QuoteMeta(`req_requesturi="/test?key=val"
	errfmt.newWithDetails() errfmt.go:0
	errfmt.GenerateDeepErrors() errfmt.go:0
	`+handleFuncShortName+`() formatter_html_test.go:0
	errfmt.HTTPHandlerWithError.func1() formatter_html.go:0
	net/http.HandlerFunc.ServeHTTP() server.go:0
	net/http.(*ServeMux).ServeHTTP() server.go:0
	net/http.serverHandler.ServeHTTP() server.go:0
	net/http.(*conn).serve() server.go:0`) + `
$`,
	}

	patternItems := strings.Join(patternInfoItems, " ") + strings.Join(patternWarningItems, " ")
	//fmt.Printf("patternItems:\n%s\n", patternItems)

	assert.Regexp(t,
		patternItems,
		logLine,
		"Log",
	)
}
