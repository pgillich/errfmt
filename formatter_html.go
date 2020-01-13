package errfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moogar0880/problems"
	log "github.com/sirupsen/logrus"
)

const (
	// KeyHTTPProblemError is Logrus details key for rendering error
	KeyHTTPProblemError = "httpproblem_error"
	// KeyHTTPWriteError is Logrus details key for response write error
	KeyHTTPWriteError = "httpwrite_error"
	// ContentTypeProblem is RFC7807 compliant HTTP ContentType
	ContentTypeProblem = "application/problem+json"
	// ContentTypeJSON is JSON Content-Type
	ContentTypeJSON = "application/json; charset=utf-8"
	// KeyPrefixRequest is the prefix of request info as logrus field
	KeyPrefixRequest = "req_"
	// KeyPrefixResponse is the prefix of response info as logrus field
	KeyPrefixResponse = "resp_"
	// KeyHTTPStatus is Logrus details key for HTTP Status
	KeyHTTPStatus = KeyPrefixResponse + "status"
	// KeyHandlerFunc is the field key to HTTP HanderFunc
	KeyHandlerFunc = "handlerFunc"
)

func NewHTTPProblemLogger(flags int, callStackSkipLast int) *log.Logger {
	buffer := &bytes.Buffer{}

	logger := &log.Logger{
		Out:          buffer,
		Hooks:        make(log.LevelHooks),
		Formatter:    NewHTTPProblemFormatter(buffer, flags&FlagCallStackInHTTPProblem > 0),
		ReportCaller: true,
		Level:        log.TraceLevel,
		ExitFunc:     os.Exit,
	}

	if flags&FlagExtractDetails > 0 {
		logger.AddHook(HookAllLevels(AppendDetailsToEntry))
	}

	return logger
}

type FormatterFunc func(*log.Entry) ([]byte, error)

func (formatter FormatterFunc) Format(entry *log.Entry) ([]byte, error) {
	return formatter(entry)
}

func NewHTTPProblemFormatter(buffer *bytes.Buffer, callStackInHTTPProblem bool) log.Formatter {
	return FormatterFunc(func(entry *log.Entry) ([]byte, error) {
		buffer.Reset()

		statusCode := http.StatusInternalServerError

		if status, has := entry.Data[KeyHTTPStatus]; has {
			switch s := status.(type) {
			case int:
				statusCode = s
			}
		}

		return RenderHTTPProblem2(statusCode, entry, callStackInHTTPProblem)
	})
}

// HTTPHandlerWithLoggerFunc is an extended http.HandlerFunc with logger
type HTTPHandlerWithLoggerFunc func(w http.ResponseWriter, r *http.Request, logger *log.Logger)

/*HTTPHandlerWithLogger decorates HTTPHandlerWithLoggerFunc to http.HandlerFunc
It's a simple decorator function to pass a prepared logger to a http.HandlerFunc implementation
*/
func HTTPHandlerWithLogger(handler HTTPHandlerWithLoggerFunc, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, logger)
	}
}

// HTTPErrorHandler is an error and log handler for HTTP responses
type HTTPErrorHandler struct {
	*log.Logger
	LevelByStatus map[int]log.Level
}

// GetLogLevelByStatus returns the log level by the first digit of HTTP status
// the default value is TraceLevel
func (handler *HTTPErrorHandler) GetLogLevelByStatus(status int) log.Level {
	if level, ok := handler.LevelByStatus[status/100]; ok {
		return level
	}

	return log.TraceLevel
}

// DefaultLevelByStatus returns an example for different log levels by HTTP Status
func DefaultLevelByStatus() map[int]log.Level {
	return map[int]log.Level{
		2: log.DebugLevel,
		4: log.WarnLevel,
		5: log.ErrorLevel,
	}
}

// HTTPHandlerWithErrorFunc is an extended http.HandlerFunc with logger and error handling
type HTTPHandlerWithErrorFunc func(w http.ResponseWriter, r *http.Request,
	logger *log.Logger,
) (jsonObj interface{}, status int, err error)

/*HTTPHandlerWithError decorates HTTPHandlerWithErrorFunc to http.HandlerFunc

if HTTPHandlerWithErrorFunc returns nil error, the HTTPHandlerWithErrorFunc ...:
	- MAY: HTTP Content-Type (optional, http.ResponseWriter.Header)
	- MUST: response status (http.ResponseWriter.WriteHeader)
	- MUST: response body (http.ResponseWriter.Write)

if HTTPHandlerWithErrorFunc returns NOT nil error, it ...:
	- HTTP Content-Type is overwritten to application/problem+json
	- problem response body is built and sent, conform to RFT7807
	- http.ResponseWriter.WriteHeader is called automatically
	- log.Warning is called to print error message on console
*/
func HTTPHandlerWithError(handler HTTPHandlerWithErrorFunc,
	logger *log.Logger, levelByStatus map[int]log.Level,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var status int
		var err, errWrite error
		var jsonObj interface{}

		handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
		errorHandler := HTTPErrorHandler{
			Logger:        logger,
			LevelByStatus: levelByStatus,
		}

		jsonObj, status, err = handler(w, r, logger)

		logLevel := errorHandler.GetLogLevelByStatus(status)

		entry := logger.WithField(KeyHandlerFunc, handlerName)
		if err == nil {
			var jsonBytes []byte

			jsonBytes, err = json.MarshalIndent(jsonObj, "", "  ")
			if err == nil {
				if w.Header().Get("Content-Type") == "" {
					w.Header().Add("Content-Type", ContentTypeJSON)
				}

				w.WriteHeader(status)

				_, errWrite = w.Write(jsonBytes)
				if errWrite != nil {
					entry.Data[KeyHTTPWriteError] = errWrite.Error()

					if logLevel > log.ErrorLevel {
						logLevel = log.ErrorLevel
					}
				}
			}
		}

		if err != nil {
			entry = entry.WithError(err)
			entry = WriteHTTPProblem(w, status, entry)
		}

		entry.Log(logLevel)
	}
}

/*RequestInfoHook implements logrus.Hook
The method, host, remoteaddr and requesturi are defined on http.Request,
other keys are HTTP header fields.
*/
type RequestInfoHook struct {
	request    *http.Request
	infoFields []string
}

// AddHookRequestInfo makes and registers a new hook to logger
func AddHookRequestInfo(logger *log.Logger, request *http.Request, infoFields []string) {
	hook := &RequestInfoHook{
		request:    request,
		infoFields: infoFields,
	}
	logger.AddHook(hook)
}

// Fire is called by logrus.Entry.log(), which is thread-safe, see there.
func (hook *RequestInfoHook) Fire(entry *log.Entry) error {
	hook.appendRequestInfo(entry)

	return nil
}

// Levels returns all levels
func (hook *RequestInfoHook) Levels() []log.Level {
	return log.AllLevels
}

// appendRequestInfo adds HTTP connection and header info to entry
func (hook *RequestInfoHook) appendRequestInfo(entry *log.Entry) {
	appendRequestInfo(entry, hook.request, hook.infoFields)
}

// appendRequestInfo adds HTTP connection and header info to entry
func appendRequestInfo(entry *log.Entry, request *http.Request, infoFields []string) {
	for _, field := range infoFields {
		switch field {
		case "method":
			entry.Data[KeyPrefixRequest+field] = request.Method

		case "host":
			entry.Data[KeyPrefixRequest+field] = request.Host

		case "remoteaddr":
			entry.Data[KeyPrefixRequest+field] = request.RemoteAddr

		case "requesturi":
			entry.Data[KeyPrefixRequest+field] = request.RequestURI

		default:
			if value := request.Header.Get(field); value != "" {
				entry.Data[FixStructuredDataName(KeyPrefixRequest+field)] = value
			}
		}
	}
}

// DefaultSelectedRequestInfo returns most informative request fields
func DefaultSelectedRequestInfo() []string {
	return []string{
		"method", "host", "remoteaddr", "requesturi",
		"From", "Forwarded", "Content-Length",
		"X-Forwarded-For", "X-Forwarded-Host", "X-Http-Method-Override",
	}
}

type GinHandlerWithErrorFunc func(c *gin.Context,
	logger *log.Logger,
) (jsonObj interface{}, status int, err error)

func GinHandlerWithError(handler GinHandlerWithErrorFunc,
	logger *log.Logger, levelByStatus map[int]log.Level,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var jsonObj interface{}
		var status int
		var err error

		handlerName := c.HandlerName()
		errorHandler := HTTPErrorHandler{
			Logger:        logger,
			LevelByStatus: levelByStatus,
		}

		jsonObj, status, err = handler(c, logger)

		entry := logger.WithField(KeyHandlerFunc, handlerName)
		if err != nil {
			entry = entry.WithError(err)
			jsonObj = BuildHTTPProblem(status, entry)
			c.Header("Content-Type", ContentTypeProblem)
		}

		c.IndentedJSON(status, jsonObj)

		entry.Log(errorHandler.GetLogLevelByStatus(status))
	}
}

/*WriteHTTPProblem sends a HTTP problem response.
- Sets response Content-Type to application/problem+json
- Sets response status code
- Builds and writes problem body (JSON)
- Returns entry extended by body build error, if any (conforming to Fluent Builder pattern)
*/
func WriteHTTPProblem(w http.ResponseWriter, statusCode int, entry *log.Entry) *log.Entry {
	respBody := []byte{}

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(statusCode)

	entry = ExtractHTTPProblem(&respBody, statusCode, entry)
	if _, err := w.Write(respBody); err != nil {
		entry.Data[KeyHTTPWriteError] = err.Error()
	}

	return entry
}

/*ExtractHTTPProblem builds a HTTP problem body from entry
- Renders problem body (JSON)
- Returns entry extended by body build error, if any (conforming to Fluent Builder pattern)
*/
func ExtractHTTPProblem(respBody *[]byte, statusCode int, entry *log.Entry) *log.Entry {
	body, err := RenderHTTPProblem(statusCode, entry)
	if err != nil {
		entry.Data[KeyHTTPProblemError] = err.Error()
	}
	*respBody = body

	return entry
}

/*GetAdvancedFormatter returns the AdvancedFormatter part
Returns nil, if AdvancedFormatter does not exists
*/
func GetAdvancedFormatter(formatter log.Formatter) *AdvancedFormatter {
	switch f := formatter.(type) {
	case *AdvancedTextFormatter:
		return &f.AdvancedFormatter
	case *AdvancedSyslogFormatter:
		return &f.AdvancedFormatter
	case *AdvancedJSONFormatter:
		return &f.AdvancedFormatter
	}

	return nil
}

// BuildHTTPProblem builds a new HTTPProblem instance
// It's the worker function of HTTP problem response
// nolint:golint,gocyclo,funlen
func BuildHTTPProblem(statusCode int, entry *log.Entry) *HTTPProblem {
	f := GetAdvancedFormatter(entry.Logger.Formatter)
	data := f.PrepareFields(entry, GetClashingFieldsHTTP())

	if entry.Time.IsZero() {
		data[log.FieldKeyTime] = time.Now().Format(time.RFC3339)
	} else {
		data[log.FieldKeyTime] = entry.Time.Format(time.RFC3339)
	}

	callStack := []string{}
	callStackLines := f.GetCallStack(entry)

	if (f.Flags & FlagCallStackInHTTPProblem) > 0 {
		callStack = callStackLines
	}

	title := http.StatusText(statusCode)

	details := map[string]string{}

	for k, v := range data {
		bytes, err := JSONMarshal(v, "", false)

		var jsonValue string
		if err != nil {
			jsonValue = err.Error()
		} else {
			jsonValue = string(bytes)
		}
		details[k] = jsonValue
	}

	detail := ""
	if err := GetError(entry); err != nil {
		detail = err.Error()
	} else if msg, ok := data[log.FieldKeyMsg]; ok {
		detail = fmt.Sprintf("%s", msg)
	}

	return NewHTTPProblem(
		statusCode,
		title,
		detail,
		details,
		callStack,
	)
}

func BuildHTTPProblem2(statusCode int, entry *log.Entry, callStackInHTTPProblem bool) *HTTPProblem {
	PrepareFields(entry, GetClashingFieldsHTTP())

	if entry.Time.IsZero() {
		entry.Data[log.FieldKeyTime] = time.Now().Format(time.RFC3339)
	} else {
		entry.Data[log.FieldKeyTime] = entry.Time.Format(time.RFC3339)
	}

	callStack := []string{}
	if callStackInHTTPProblem {
		callStack = GetCallStack(entry)
	}

	title := http.StatusText(statusCode)

	details := map[string]string{}

	for k, v := range entry.Data {
		bytes, err := JSONMarshal(v, "", false)

		var jsonValue string
		if err != nil {
			jsonValue = err.Error()
		} else {
			jsonValue = string(bytes)
		}
		details[k] = jsonValue
	}

	detail := ""
	if err := GetError(entry); err != nil {
		detail = err.Error()
	} else if msg, ok := entry.Data[log.FieldKeyMsg]; ok {
		detail = fmt.Sprintf("%s", msg)
	}

	return NewHTTPProblem(
		statusCode,
		title,
		detail,
		details,
		callStack,
	)
}

// RenderHTTPProblem renders HTTPProblem a JSON
func RenderHTTPProblem(statusCode int, entry *log.Entry) ([]byte, error) {
	httpProblem := BuildHTTPProblem(statusCode, entry)

	resp, err := JSONMarshal(httpProblem, "  ", false)
	if err != nil {
		httpProblem = NewHTTPProblem(
			http.StatusInternalServerError,
			http.StatusText(http.StatusInternalServerError),
			err.Error(),
			map[string]string{},
			[]string{},
		)

		resp, _ = JSONMarshal(httpProblem, "  ", false) // nolint:errcheck
	}

	return resp, err
}

func RenderHTTPProblem2(statusCode int, entry *log.Entry, callStackInHTTPProblem bool) ([]byte, error) {
	httpProblem := BuildHTTPProblem2(statusCode, entry, callStackInHTTPProblem)

	resp, err := JSONMarshal(httpProblem, "  ", false)
	if err != nil {
		httpProblem = NewHTTPProblem(
			http.StatusInternalServerError,
			http.StatusText(http.StatusInternalServerError),
			err.Error(),
			map[string]string{},
			[]string{},
		)

		resp, _ = JSONMarshal(httpProblem, "  ", false) // nolint:errcheck
	}

	return resp, err
}

// HTTPProblem is RFC-7807 comliant response
type HTTPProblem struct {
	problems.DefaultProblem
	Details   map[string]string `json:"details,omitempty"`
	CallStack []string          `json:"callstack,omitempty"`
}

// NewHTTPProblem makes a HTTPProblem instance
func NewHTTPProblem(status int, title string, message string,
	details map[string]string, callStack []string,
) *HTTPProblem {
	p := HTTPProblem{
		DefaultProblem: problems.DefaultProblem{
			Type:   problems.DefaultURL,
			Title:  title,
			Status: status,
			Detail: message,
		},
		Details:   details,
		CallStack: callStack,
	}

	return &p
}

// GetClashingFieldsHTTP returns the automatical filles fields
func GetClashingFieldsHTTP() []string {
	return []string{
		log.FieldKeyTime, log.FieldKeyFunc,
		log.FieldKeyMsg, log.FieldKeyFile, KeyCallStack,
	}
}
