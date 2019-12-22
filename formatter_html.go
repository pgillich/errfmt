package errfmt

import (
	"fmt"
	"net/http"
	"time"

	"github.com/moogar0880/problems"
	log "github.com/sirupsen/logrus"
)

const (
	// nolint:golint
	KeyHTTPProblemError = "httpproblem_error"
	// nolint:golint
	ContentTypeProblem = "application/problem+json"
)

// nolint:golint
func WriteHTTPProblem(w http.ResponseWriter, statusCode int, entry *log.Entry) *log.Entry {
	respBody := []byte{}

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(statusCode)

	entry = ExtractHTTPProblem(&respBody, statusCode, entry)
	if _, err := w.Write(respBody); err != nil {
		entry.Data[KeyHTTPProblemError] = err
	}

	return entry
}

// nolint:golint
func ExtractHTTPProblem(respBody *[]byte, statusCode int, entry *log.Entry) *log.Entry {
	body, err := RenderHTTPProblem(statusCode, entry)
	if err != nil {
		entry.Data[KeyHTTPProblemError] = err
	}
	*respBody = body

	return entry
}

// GetAdvancedFormatter returns the AdvancedFormatter part
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

// BuildHTTPProblem builds a new HTTPProblem
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
	// TODO cleanup
	// nolint:gocritic
	/*
		if errorVal, ok := data[log.ErrorKey]; ok {
			if err, ok := errorVal.(error); ok {
				title = DigErrorsString(err)
			}
		}
	*/

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
	if err := f.GetError(entry); err != nil {
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
