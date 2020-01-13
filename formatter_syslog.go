package errfmt

import (
	"fmt"
	"strings"

	"github.com/juju/rfc/rfc5424"
	log "github.com/sirupsen/logrus"
)

const (
	// nolint:golint
	StructuredIDDetails = "details"
	// nolint:golint
	StructuredIDCallStack = "calls"
)

// nolint:golint
func NewSyslogLogger(level log.Level, flags int, callStackSkipLast int,
	facility rfc5424.Facility, hostname rfc5424.Hostname, appName string,
	procID string, msgID string,
) *log.Logger {
	logger := log.New()

	logger.Formatter = NewAdvancedSyslogFormatter(flags, callStackSkipLast,
		facility, hostname, appName, procID, msgID)
	logger.Level = level
	logger.ReportCaller = true

	if flags&FlagExtractDetails > 0 {
		logger.AddHook(HookAllLevels(AppendDetailsToEntry))
	}

	if flags&FlagCallStackInFields > 0 {
		logger.AddHook(HookAllLevels(AppendCallStackToEntry(callStackSkipLast)))
	}

	if flags&FlagPrintStructFieldNames > 0 {
		logger.AddHook(HookAllLevels(RenderStructFieldNames))
	}

	return logger
}

// nolint:golint
type AdvancedSyslogFormatter struct {
	LevelToSeverity map[log.Level]rfc5424.Severity
	Facility        rfc5424.Facility
	Hostname        rfc5424.Hostname
	AppName         rfc5424.AppName
	ProcID          rfc5424.ProcID
	MsgID           rfc5424.MsgID
	ConsoleFlags
	AdvancedFormatter
	SortingFunc func([]string)
}

// nolint:golint
func NewAdvancedSyslogFormatter(flags int, callStackSkipLast int,
	facility rfc5424.Facility, hostname rfc5424.Hostname, appName string,
	procID string, msgID string,
) *AdvancedSyslogFormatter {
	advancedSyslogFormatter := AdvancedSyslogFormatter{
		LevelToSeverity: DefaultLevelToSeverity(),
		Facility:        facility,
		Hostname:        hostname,
		AppName:         rfc5424.AppName(appName),
		ProcID:          rfc5424.ProcID(procID),
		MsgID:           rfc5424.MsgID(msgID),
		ConsoleFlags: ConsoleFlags{
			CallStackOnConsole: flags&FlagCallStackOnConsole > 0,
			CallStackSkipLast:  callStackSkipLast,
		},
		AdvancedFormatter: AdvancedFormatter{
			Flags:              flags,
			CallStackSkipLastX: callStackSkipLast,
		},
		SortingFunc: SortingFuncDecorator(AdvancedFieldOrder()),
	}

	return &advancedSyslogFormatter
}

// Format implements logrus.Formatter interface
func (f *AdvancedSyslogFormatter) Format(entry *log.Entry) ([]byte, error) { //nolint:funlen,gocyclo
	var consoleCallStackLines []string

	if f.CallStackOnConsole {
		consoleCallStackLines = GetCallStack(entry)
	}

	PrepareFields(entry, GetClashingFieldsSyslog())

	// consoleCallStackLines cannot be dig anymore
	RenderErrorInEntry(entry)

	detailList := NewJSONDataElement(StructuredIDDetails)
	detailKeys := []string{}

	var hasKeyCallStack bool
	for key := range entry.Data {
		if key == KeyCallStack {
			hasKeyCallStack = true
		} else {
			detailKeys = append(detailKeys, key)
		}
	}

	trimJSONDquote := (f.Flags & FlagTrimJSONDquote) > 0
	f.SortingFunc(detailKeys)
	for _, key := range detailKeys {
		detailList.Append(key, entry.Data[key], trimJSONDquote)
	}

	structuredData := rfc5424.StructuredData{
		detailList,
	}

	msgIDdefault := "DETAILS_MSG"
	if hasKeyCallStack {
		msgIDdefault = "DETAILS_CALLS_MSG"

		callsList := NewJSONDataElement(StructuredIDCallStack)
		callsList.Append(KeyCallStack, entry.Data[KeyCallStack], trimJSONDquote)

		structuredData = append(structuredData, callsList)
	}

	msgID := f.MsgID
	if msgID == "" {
		msgID = rfc5424.MsgID(msgIDdefault)
	}

	message := rfc5424.Message{
		Header: rfc5424.Header{
			Priority: rfc5424.Priority{
				Severity: f.LevelToSeverity[entry.Level],
				Facility: f.Facility,
			},
			Timestamp: rfc5424.Timestamp{Time: entry.Time},
			Hostname:  f.Hostname,
			AppName:   f.AppName,
			ProcID:    f.ProcID,
			MsgID:     msgID,
		},
		StructuredData: structuredData,
		Msg:            entry.Message,
	}

	textPart := []byte(MessageString(message))

	if len(consoleCallStackLines) > f.CallStackSkipLast {
		textPart = AppendCallStack(textPart, consoleCallStackLines[:len(consoleCallStackLines)-f.CallStackSkipLast])
	}

	return textPart, nil
}

// nolint:golint
func MessageString(m rfc5424.Message) string {
	stringStructuredData := StructuredDataString(m.StructuredData)

	if m.Msg == "" {
		return fmt.Sprintf("%s %s", m.Header, stringStructuredData)
	}

	return fmt.Sprintf("%s %s %s", m.Header, stringStructuredData, m.Msg)
}

// nolint:golint
func StructuredDataString(sd rfc5424.StructuredData) string {
	if len(sd) == 0 {
		return "-"
	}

	elems := make([]string, len(sd))
	for i, elem := range sd {
		elems[i] = StructuredDataElementString(elem)
	}

	return strings.Join(elems, "")
}

// nolint:golint
func StructuredDataElementString(sde rfc5424.StructuredDataElement) string {
	params := sde.Params()
	if len(params) == 0 {
		return fmt.Sprintf("[%s]", sde.ID())
	}

	paramStrs := make([]string, len(params))
	for i, param := range params {
		paramStrs[i] = StructuredDataParamSting(param)
	}

	return fmt.Sprintf("[%s %s]", sde.ID(), strings.Join(paramStrs, " "))
}

// nolint:golint
func StructuredDataParamSting(sdp rfc5424.StructuredDataParam) string {
	return fmt.Sprintf(`%s="%s"`, sdp.Name, sdp.Value)
}

// nolint:golint
func DefaultLevelToSeverity() map[log.Level]rfc5424.Severity {
	return map[log.Level]rfc5424.Severity{
		log.PanicLevel: rfc5424.SeverityAlert,
		log.FatalLevel: rfc5424.SeverityCrit,
		log.ErrorLevel: rfc5424.SeverityError,
		log.WarnLevel:  rfc5424.SeverityWarning,
		log.InfoLevel:  rfc5424.SeverityNotice,
		log.DebugLevel: rfc5424.SeverityInformational,
		log.TraceLevel: rfc5424.SeverityDebug,
	}
}

// nolint:golint
type JSONDataElement struct {
	id     string
	params []rfc5424.StructuredDataParam
}

// nolint:golint
func NewJSONDataElement(id string) *JSONDataElement {
	return &JSONDataElement{id: id}
}

// nolint:golint
func (de *JSONDataElement) Append(name string, value interface{}, trimJSONDquote bool) {
	bytes, err := JSONMarshal(value, "", false)

	var jsonValue string
	if err != nil {
		jsonValue = err.Error()
	} else {
		jsonValue = string(bytes)
	}

	if trimJSONDquote && strings.HasPrefix(jsonValue, `"`) && strings.HasSuffix(jsonValue, `"`) {
		jsonValue = jsonValue[1 : len(jsonValue)-1]
	}

	sdp := rfc5424.StructuredDataParam{
		Name:  rfc5424.StructuredDataName(FixStructuredDataName(name)),
		Value: rfc5424.StructuredDataParamValue(jsonValue),
	}
	de.params = append(de.params, sdp)
}

// nolint:golint
func (de *JSONDataElement) ID() rfc5424.StructuredDataName {
	return rfc5424.StructuredDataName(de.id)
}

// nolint:golint
func (de *JSONDataElement) Params() []rfc5424.StructuredDataParam {
	return de.params
}

// nolint:golint
func (de *JSONDataElement) Validate() error { return nil }

// nolint:golint
func FixStructuredDataName(name string) string {
	str := strings.Builder{}

	for _, b := range []byte(name) {
		if b < '!' || b > '~' || b == '=' || b == ' ' || b == ']' || b == '"' { // SD-NAME
			str.WriteByte('_') //nolint:gosec
		} else {
			str.WriteByte(b) //nolint:gosec
		}
	}

	return str.String()
}

func GetClashingFieldsSyslog() []string {
	return []string{
		log.FieldKeyLevel, log.FieldKeyTime, log.FieldKeyFunc,
		log.FieldKeyMsg, log.FieldKeyFile, // KeyCallStack,
	}
}

// prefixFieldClashes is a copy of Logrus feature
func prefixFieldClashes(data log.Fields, key string) {
	if v, ok := data[key]; ok {
		data["fields."+key] = v
		delete(data, key)
	}
}
