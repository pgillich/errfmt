# errorformatter

Error formatter is a Golang library for formatting logrus + emperror/errors messages.

## Introduction

The objective of this library is printing readable and parseable log messages, using another error/log handling libraries, instead of writing a new error/log library from scratch. There is no perfect solution: it's a tradeoff.

## Features

### WithErrorDetailsCallStack

If CallStackSkipLast > 0, a shorten call stack is printed. Below options don't have effect, if CallStackSkipLast = 0.

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

## Syslog

It's an RFC5424 logger, using Logrus and <https://godoc.org/github.com/juju/rfc/rfc5424>.

*PARTLY IMPLEMENTED*

## TextLogger

The output is a customized Logrus TextFormatter.

## JSONLogger

The output is a customized Logrus JSONFormatter.

Related pull requests:

* Allow disabling new line appending for json_formatter <https://github.com/sirupsen/logrus/pull/674>

## HTTPProblem

It's a RFC7807 response builder, based on Logrus and github.com/moogar0880/problems

*NOT IMPLEMENTED*

## Logrus pull requests

Unfortunately, sirupsen/logrus/Entry.log() always overwrites Entry.Caller, instead of leaving the patched value, if it's not nil.

Related pull requests:

* add caller skip <https://github.com/sirupsen/logrus/pull/973>
