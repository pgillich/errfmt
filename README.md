# errorformatter

`errorformatter` is a Golang library for formatting logrus + emperror/errors messages.

## Introduction

The objective of this library is printing readable and parseable log messages, using another error/log handling libraries, instead of writing a new error/log library from scratch. There is no perfect solution: it's a tradeoff.

## Usage

Usage examples are written and printed out by <https://github.com/pgillich/logtester>.

### Simple Info log

Here is the most simple usage example:

```go
import (
	"github.com/pgillich/errorformatter"
	log "github.com/sirupsen/logrus"
)

func trySampleText() {
	// register a trim prefix (optional)
	errorformatter.AddSkipPackageFromStackTrace("github.com/pgillich/logtester")

	// build a new logrus logger
	logger := errorformatter.NewTextLogger(log.InfoLevel, 0, 0)

	// Info log with key-value map
	logger.WithFields(log.Fields{
		"STR":  "str",
		"INT":  42,
		"BOOL": true,
	}).Info("USER MSG")
}
```

The output of above code, runing `./logtester errorformatter --testCase sampletext`

```log
level=info time="2019-10-15T22:32:25+02:00" func=errorformatter_tester.trySampleText msg="USER MSG" file="sample_text.go:21" BOOL=true INT=42 STR=str
```

Differences to original logrus logger:

* field order is similar to syslog ([RFC5424](https://tools.ietf.org/html/rfc5424))
* logrus Logger.ReportCaller is true by default
* package prefix trimmed from the func name and file path

### HTTP problem handler

It's a RFC7807 response builder, based on Logrus and github.com/moogar0880/problems. Example for using in func decorator:

```go
package errorformatter_tester

import (
	"net/http"

	"github.com/pgillich/errorformatter"
	log "github.com/sirupsen/logrus"
)

func trySampleHTTP() {
	// register a trim prefix (optional)
	errorformatter.AddSkipPackageFromStackTrace("github.com/pgillich/logtester")

	// build a new logrus logger
	logger := errorformatter.NewTextLogger(log.InfoLevel, 0, 0)

	// this func decorator sets body, header and status, if response error is NOT nil
	handler := func(w http.ResponseWriter, r *http.Request) {
		if statusCode, err := doRequest(w, r); err != nil { // calling worker func
			errorformatter.WriteHTTPProblem( // sending HTTP error
				w, logger.WithError(err), log.ErrorLevel, statusCode, // HTTP error parameters
			).Log(log.ErrorLevel, "USER MSG") // logging to the console
		}
	}

	// register the decorated handler
	mux.HandleFunc("/api", handler)
}

/*
doRequest makes the main part of the request.
	if the returned error is nil, the response body, header and status is set
	if the returned error is NOT nil, the response body and status is NOT set (the caller must do it)
*/
// nolint:unparam
func doRequest(w http.ResponseWriter, r *http.Request) (int, error) {
	return http.StatusPreconditionFailed, errorformatter.GenerateDeepErrors()
}
```

The output of above code, runing `./logtester errorformatter --testCase samplehttp`

```log
level=error time="2019-10-15T22:51:27+02:00" func=errorformatter_tester.trySampleHTTP.func1 error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="sample_http.go:26"
412
application/problem+json
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T22:51:27+02:00\""
  }
}
```

The `errorformatter.WriteHTTPProblem()` func writes the HTTP error response. So, error logging into the console and sending HTTP error can be written into one line. For more specific use case, `errorformatter.ExtractHTTPProblem()` can be used.

## Advanced error handling

`errorformatter` supports below console formatters:

The output is a customized Logrus TextFormatter:

```go
func NewTextLogger(level log.Level, flags int, callStackSkipLast int,
) *log.Logger
```

It's an RFC5424 logger, using Logrus and <https://godoc.org/github.com/juju/rfc/rfc5424>:

```go
func NewSyslogLogger(level log.Level, flags int, callStackSkipLast int,
	facility rfc5424.Facility, hostname rfc5424.Hostname, appName string,
	procID string, msgID string,
) *log.Logger
```

The output is a customized Logrus JSONFormatter:

```go
func NewJSONLogger(level log.Level, flags int, callStackSkipLast int,
) *log.Logger
```

Related pull request: allow disabling new line appending for json_formatter <https://github.com/sirupsen/logrus/pull/674>

Where:

* level: the logger level
* flags: error formatting
  * FlagNone: no any flag is active
  * FlagExtractDetails: extracts errors.Details to logrus.Fields
  * FlagCallStackInFields: extracts errors.StackTrace() to logrus.Fields
  * FlagCallStackOnConsole: extracts errors.StackTrace() to logrus.Field "callstack"
  * FlagCallStackInHTTPProblem: extracts errors.StackTrace() to HTTPProblem
  * FlagPrintStructFieldNames: renders non-scalar Details values are by "%+v", instead of "%v"
  * FlagTrimJSONDquote: trims the leading and trailing '"' of JSON-formatted values
* callStackSkipLast: skipping last lines from the call stack
* facility: Syslog facility
* hostname: Syslog hostname
* appName: Syslog appname
* procID: Syslog Proc ID
* msgID: Syslog message format ID

Example for using formatting flags and callStackSkipLast:

```go
    logger := errorformatter.NewTextLogger(log.InfoLevel,
        FlagExtractDetails|FlagCallStackOnConsole|FlagPrintStructFieldNames,
        2)

    (...)

    logger.WithError(err).Log(log.ErrorLevel, "USER MSG")

    logger.WithError(err).Error("USER MSG")
```

The `WithError(error)` must be called on the logger.

## Format flags

Effect of error format flags can be tested with `logtester`. In most cases, the outputs are:

1. Text formatter
1. Syslog formatter
1. HTTP error response

HTTP error response is also printed.

Error details and messages are generated from below source code:

```go
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
```

### FlagNone

It's the baseline

```shell
./logtester errorformatter --testCase errorhttp --formatter text
./logtester errorformatter --testCase errorhttp --formatter syslog
```

```log
level=error time="2019-10-15T23:37:54+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61"
```

```log
<27>1 2019-10-15T23:39:33.303198568+02:00 fqdn.host.com application PID DETAILS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\""] USER MSG
```

```log
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T23:37:54+02:00\""
  }
}
```

### FlagExtractDetails

```shell
./logtester errorformatter --testCase errorhttp --formatter text --extractDetails
./logtester errorformatter --testCase errorhttp --formatter syslog --extractDetails
```

```log
level=error time="2019-10-15T23:40:27+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
```

```log
<27>1 2019-10-15T23:41:20.585905377+02:00 fqdn.host.com application PID DETAILS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\"" K0_1="\"V0_1\"" K0_2="\"V0_2\"" K1_1="\"V1_1\"" K1_2="\"V1_2\"" K3_2="\"V3 space\"" K3_5="\"V3\\\"doublequote\"" K3%6="\"V3%percent\"" K3:3="\"V3:column\"" K3;3="\"V3;semicolumn\"" K3_1="\"V3=equal\"" K5_bool="true" K5_int="12" K5_map="{\"1\":\"ONE\",\"2\":\"TWO\"}" K5_struct="{\"Text\":\"text\",\"Integer\":42,\"Bool\":true}"] USER MSG
```

```log
{
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
    "time": "\"2019-10-15T23:40:27+02:00\""
  }
}
```

### FlagCallStackInFields

```shell
./logtester errorformatter --testCase errorhttp --formatter text --callStackInFields
./logtester errorformatter --testCase errorhttp --formatter syslog --callStackInFields
```

```log
level=error time="2019-10-15T23:43:08+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61" callstack="[github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295 github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271 errorformatter_tester.tryErrorHTTP() test_formatter.go:55 errorformatter_tester.TryErrorformatter() test_formatter.go:47 cmd.testErrorformatter() errorformatter.go:115 cmd.glob..func1() errorformatter.go:44 github.com/spf13/cobra.(*Command).execute() command.go:830 github.com/spf13/cobra.(*Command).ExecuteC() command.go:914 github.com/spf13/cobra.(*Command).Execute() command.go:864 cmd.Execute() zz_root.go:25 main.main() main.go:8 runtime.main() proc.go:203 runtime.goexit() asm_amd64.s:1357]"
```

```log
<27>1 2019-10-15T23:42:14.739382013+02:00 fqdn.host.com application PID DETAILS_CALLS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\""][calls callstack="[\"github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295\",\"github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271\",\"errorformatter_tester.tryErrorHTTP() test_formatter.go:55\",\"errorformatter_tester.TryErrorformatter() test_formatter.go:47\",\"cmd.testErrorformatter() errorformatter.go:115\",\"cmd.glob..func1() errorformatter.go:44\",\"github.com/spf13/cobra.(*Command).execute() command.go:830\",\"github.com/spf13/cobra.(*Command).ExecuteC() command.go:914\",\"github.com/spf13/cobra.(*Command).Execute() command.go:864\",\"cmd.Execute() zz_root.go:25\",\"main.main() main.go:8\",\"runtime.main() proc.go:203\",\"runtime.goexit() asm_amd64.s:1357\"\]"] USER MSG
```

```log
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "callstack": "[\"github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295\",\"github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271\",\"errorformatter_tester.tryErrorHTTP() test_formatter.go:55\",\"errorformatter_tester.TryErrorformatter() test_formatter.go:47\",\"cmd.testErrorformatter() errorformatter.go:115\",\"cmd.glob..func1() errorformatter.go:44\",\"github.com/spf13/cobra.(*Command).execute() command.go:830\",\"github.com/spf13/cobra.(*Command).ExecuteC() command.go:914\",\"github.com/spf13/cobra.(*Command).Execute() command.go:864\",\"cmd.Execute() zz_root.go:25\",\"main.main() main.go:8\",\"runtime.main() proc.go:203\",\"runtime.goexit() asm_amd64.s:1357\"]",
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T23:42:14+02:00\""
  }
}
```

### FlagCallStackOnConsole

```shell
./logtester errorformatter --testCase errorhttp --formatter text --callStackOnConsole
./logtester errorformatter --testCase errorhttp --formatter syslog --callStackOnConsole
```

```log
level=error time="2019-10-15T23:44:14+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61"
	github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295
	github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271
	errorformatter_tester.tryErrorHTTP() test_formatter.go:55
	errorformatter_tester.TryErrorformatter() test_formatter.go:47
	cmd.testErrorformatter() errorformatter.go:115
	cmd.glob..func1() errorformatter.go:44
	github.com/spf13/cobra.(*Command).execute() command.go:830
	github.com/spf13/cobra.(*Command).ExecuteC() command.go:914
	github.com/spf13/cobra.(*Command).Execute() command.go:864
	cmd.Execute() zz_root.go:25
	main.main() main.go:8
	runtime.main() proc.go:203
	runtime.goexit() asm_amd64.s:1357
```

```log
<27>1 2019-10-15T23:45:21.164037629+02:00 fqdn.host.com application PID DETAILS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\""] USER MSG
	github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295
	github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271
	errorformatter_tester.tryErrorHTTP() test_formatter.go:55
	errorformatter_tester.TryErrorformatter() test_formatter.go:47
	cmd.testErrorformatter() errorformatter.go:115
	cmd.glob..func1() errorformatter.go:44
	github.com/spf13/cobra.(*Command).execute() command.go:830
	github.com/spf13/cobra.(*Command).ExecuteC() command.go:914
	github.com/spf13/cobra.(*Command).Execute() command.go:864
	cmd.Execute() zz_root.go:25
	main.main() main.go:8
	runtime.main() proc.go:203
	runtime.goexit() asm_amd64.s:1357
```

```log
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T23:44:14+02:00\""
  }
}
```

### FlagCallStackInHTTPProblem

```shell
/logtester errorformatter --testCase errorhttp --formatter text --callStackInHTTPProblem
/logtester errorformatter --testCase errorhttp --formatter syslog --callStackInHTTPProblem
```

```log
level=error time="2019-10-15T23:47:09+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61"
```

```log
<27>1 2019-10-15T23:46:36.314737301+02:00 fqdn.host.com application PID DETAILS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\""] USER MSG
```

```log
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T23:46:36+02:00\""
  },
  "callstack": [
    "github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295",
    "github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271",
    "errorformatter_tester.tryErrorHTTP() test_formatter.go:55",
    "errorformatter_tester.TryErrorformatter() test_formatter.go:47",
    "cmd.testErrorformatter() errorformatter.go:115",
    "cmd.glob..func1() errorformatter.go:44",
    "github.com/spf13/cobra.(*Command).execute() command.go:830",
    "github.com/spf13/cobra.(*Command).ExecuteC() command.go:914",
    "github.com/spf13/cobra.(*Command).Execute() command.go:864",
    "cmd.Execute() zz_root.go:25",
    "main.main() main.go:8",
    "runtime.main() proc.go:203",
    "runtime.goexit() asm_amd64.s:1357"
  ]
}
```

### FlagPrintStructFieldNames

```shell
./logtester errorformatter --testCase errorhttp --formatter text --extractDetails --printStructFieldNames
./logtester errorformatter --testCase errorhttp --formatter syslog --extractDetails --printStructFieldNames
```

```log
level=error time="2019-10-15T23:48:17+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{Text:text Integer:42 Bool:true hidden:hidden}"
```

```log
<27>1 2019-10-15T23:49:37.955901918+02:00 fqdn.host.com application PID DETAILS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\"" K0_1="\"V0_1\"" K0_2="\"V0_2\"" K1_1="\"V1_1\"" K1_2="\"V1_2\"" K3_2="\"V3 space\"" K3_5="\"V3\\\"doublequote\"" K3%6="\"V3%percent\"" K3:3="\"V3:column\"" K3;3="\"V3;semicolumn\"" K3_1="\"V3=equal\"" K5_bool="\"true\"" K5_int="12" K5_map="\"map[1:ONE 2:TWO\]\"" K5_struct="\"{Text:text Integer:42 Bool:true hidden:hidden}\""] USER MSG
```

```log
{
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
    "K5_bool": "\"true\"",
    "K5_int": "12",
    "K5_map": "\"map[1:ONE 2:TWO]\"",
    "K5_struct": "\"{Text:text Integer:42 Bool:true hidden:hidden}\"",
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-15T23:48:17+02:00\""
  }
}
```

### FlagTrimJSONDquote

```shell
./logtester errorformatter --testCase errorhttp --formatter text --extractDetails --trimJSONDquote
./logtester errorformatter --testCase errorhttp --formatter syslog --extractDetails --trimJSONDquote
```

```log
level=error time="2019-10-15T23:52:40+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61" K0_1=V0_1 K0_2=V0_2 K1_1=V1_1 K1_2=V1_2 K3 2="V3 space" K3"5="V3\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3=1="V3=equal" K5_bool=true K5_int=12 K5_map="map[1:ONE 2:TWO]" K5_struct="{text 42 true hidden}"
```

```log
<27>1 2019-10-15T23:53:25.743483528+02:00 fqdn.host.com application PID DETAILS_MSG [details level="error" func="errorformatter_tester.tryErrorHTTP" error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax" file="test_formatter.go:61" K0_1="V0_1" K0_2="V0_2" K1_1="V1_1" K1_2="V1_2" K3_2="V3 space" K3_5="V3\\\"doublequote" K3%6="V3%percent" K3:3="V3:column" K3;3="V3;semicolumn" K3_1="V3=equal" K5_bool="true" K5_int="12" K5_map="{\"1\":\"ONE\",\"2\":\"TWO\"}" K5_struct="{\"Text\":\"text\",\"Integer\":42,\"Bool\":true}"] USER MSG
```

```log
{
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
    "time": "\"2019-10-15T23:53:25+02:00\""
  }
}
```

### CallStackSkipLast

```shell
./logtester errorformatter --testCase errorhttp --formatter text --callStackInFields --callStackInHTTPProblem --callStackOnConsole --callStackSkipLast 7
./logtester errorformatter --testCase errorhttp --formatter syslog --callStackInFields --callStackInHTTPProblem --callStackOnConsole --callStackSkipLast 7
./logtester errorformatter --testCase errorhttp --formatter json --callStackInFields --callStackInHTTPProblem --callStackOnConsole --callStackSkipLast 7
```

```log
level=error time="2019-10-16T00:04:56+02:00" func=errorformatter_tester.tryErrorHTTP error="MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax" msg="USER MSG" file="test_formatter.go:61" callstack="[github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295 github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271 errorformatter_tester.tryErrorHTTP() test_formatter.go:55 errorformatter_tester.TryErrorformatter() test_formatter.go:47 cmd.testErrorformatter() errorformatter.go:115 cmd.glob..func1() errorformatter.go:44]"
	github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295
	github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271
	errorformatter_tester.tryErrorHTTP() test_formatter.go:55
	errorformatter_tester.TryErrorformatter() test_formatter.go:47
	cmd.testErrorformatter() errorformatter.go:115
	cmd.glob..func1() errorformatter.go:44
```

```log
<27>1 2019-10-16T00:06:08.01630693+02:00 fqdn.host.com application PID DETAILS_CALLS_MSG [details level="\"error\"" func="\"errorformatter_tester.tryErrorHTTP\"" error="\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"" file="\"test_formatter.go:61\""][calls callstack="[\"github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295\",\"github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271\",\"errorformatter_tester.tryErrorHTTP() test_formatter.go:55\",\"errorformatter_tester.TryErrorformatter() test_formatter.go:47\",\"cmd.testErrorformatter() errorformatter.go:115\",\"cmd.glob..func1() errorformatter.go:44\"\]"] USER MSG
	github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295
	github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271
	errorformatter_tester.tryErrorHTTP() test_formatter.go:55
	errorformatter_tester.TryErrorformatter() test_formatter.go:47
	cmd.testErrorformatter() errorformatter.go:115
	cmd.glob..func1() errorformatter.go:44
```

```log
{"callstack":["github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295","github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271","errorformatter_tester.tryErrorHTTP() test_formatter.go:55","errorformatter_tester.TryErrorformatter() test_formatter.go:47","cmd.testErrorformatter() errorformatter.go:115","cmd.glob..func1() errorformatter.go:44"],"error":"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax","file":"test_formatter.go:61","func":"errorformatter_tester.tryErrorHTTP","level":"error","msg":"USER MSG","time":"2019-10-16T00:06:30+02:00"}
	github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295
	github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271
	errorformatter_tester.tryErrorHTTP() test_formatter.go:55
	errorformatter_tester.TryErrorformatter() test_formatter.go:47
	cmd.testErrorformatter() errorformatter.go:115
	cmd.glob..func1() errorformatter.go:44
```

```log
{
  "type": "about:blank",
  "title": "Precondition Failed",
  "status": 412,
  "detail": "MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \"NO_NUMBER\": invalid syntax",
  "details": {
    "callstack": "[\"github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295\",\"github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271\",\"errorformatter_tester.tryErrorHTTP() test_formatter.go:55\",\"errorformatter_tester.TryErrorformatter() test_formatter.go:47\",\"cmd.testErrorformatter() errorformatter.go:115\",\"cmd.glob..func1() errorformatter.go:44\"]",
    "error": "\"MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing \\\"NO_NUMBER\\\": invalid syntax\"",
    "level": "\"error\"",
    "time": "\"2019-10-16T00:04:56+02:00\""
  },
  "callstack": [
    "github.com/pgillich/errorformatter.newWithDetails() errorformatter.go:295",
    "github.com/pgillich/errorformatter.GenerateDeepErrors() errorformatter.go:271",
    "errorformatter_tester.tryErrorHTTP() test_formatter.go:55",
    "errorformatter_tester.TryErrorformatter() test_formatter.go:47",
    "cmd.testErrorformatter() errorformatter.go:115",
    "cmd.glob..func1() errorformatter.go:44"
  ]
}
```

## TODO

### Entry.Caller

Unfortunately, sirupsen/logrus/Entry.log() always overwrites Entry.Caller, instead of leaving the patched value, if it's not nil.

Related pull requests:

* add caller skip <https://github.com/sirupsen/logrus/pull/973>

### Fluentd

<https://github.com/evalphobia/logrus_fluent>

Error, struct and map conversions:
<https://github.com/evalphobia/logrus_fluent/pull/32/files>
