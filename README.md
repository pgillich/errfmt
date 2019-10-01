# errorformatter

Error formatter is a Golang library for formatting logrus + emperror/errors messages.

## Introduction

The objective of this library is printing readable and parseable log messages, using another error/log handling libraries, instead of writing a new error/log library from scratch. There is no perfect solution: it's a tradeoff.

## Features

### ParentCallerHook

Patches the "func" and "file" fields to the parent.

Unfortunately, sirupsen/logrus/Entry.log() always overwrites Entry.Caller, instead of leaving the patched value, if it's not nil. The workaround is registering hooks for each log level, in order to patch the fields before calling Format()

Similar pull requests:

* add caller skip <https://github.com/sirupsen/logrus/pull/973>
* Pragma Caller hook <https://github.com/sirupsen/logrus/pull/1002>

### ErrorWithCallStack

If CallStackSkipLast > 0, a shorten call stack is printed. Below options don't have effect, if CallStackSkipLast = false.

#### CallStackSkipLast

The last CallStackSkipLast-th lines are skipped, so the main() will never be printed.

#### CallStackNewLines

If CallStackNewLines = true, call stack is printed in new lines, indented by '\t'. Not Implemented by JSONLogger.

#### CallStackInFields

If CallStackNewLines = false, call stack is appended to Details with field "callstack".

### ModuleCallerPrettyfier

Trims registered package name(s). Fits to Logrus TextFormatter.CallerPrettyfier

Similar pull requests:

* add skipPackageNameForCaller <https://github.com/sirupsen/logrus/pull/989>

### PrintStructFieldNames

If it's true, non-scalar Details values are rendered by "%+v", instead of "%v".

Related pull requests:

* Format full values for fields <https://github.com/sirupsen/logrus/pull/505>

## TextLogger

The output is a customized Logrus TextFormatter.

## JSONLogger

The output is a customized Logrus JSONFormatter.

Related pull requests:

* Allow disabling new line appending for json_formatter <https://github.com/sirupsen/logrus/pull/674>

## Syslog

It's an RFC5424 logger, based on Logrus and <https://godoc.org/github.com/juju/rfc/rfc5424>.

*NOT IMPLEMENTED*

## HTTPProblem

It's a RFC7807 response builder, based on Logrus and github.com/moogar0880/problems

*NOT IMPLEMENTED*
