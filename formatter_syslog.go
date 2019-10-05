package errorformatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/juju/rfc/rfc5424"
	log "github.com/sirupsen/logrus"
)

const (
	// nolint:golint
	StructuredIDDetails = "details"
	// nolint:golint
	StructuredIDCallStack = KeyCallStack
)

// nolint:golint
func NewSyslogLogger(level log.Level, callStackSkipLast int, callStackNewLines bool,
	facility rfc5424.Facility, hostname rfc5424.Hostname, appName string,
	procID string, msgID string,
) *AdvancedLogger {
	return &AdvancedLogger{
		Logger: &log.Logger{
			Formatter: NewAdvancedSyslogFormatter(
				facility, hostname, appName, procID, msgID, true),
			Level:        level,
			ReportCaller: true,
		},
		CallStackSkipLast: callStackSkipLast,
		CallStackNewLines: callStackNewLines,
	}
}

// nolint:golint
type AdvancedSyslogFormatter struct {
	LevelToSeverity map[log.Level]rfc5424.Severity
	Facility        rfc5424.Facility
	Hostname        rfc5424.Hostname
	AppName         rfc5424.AppName
	ProcID          rfc5424.ProcID
	MsgID           rfc5424.MsgID
	TrimJSONDquote  bool
}

// nolint:golint
func NewAdvancedSyslogFormatter(
	facility rfc5424.Facility, hostname rfc5424.Hostname, appName string,
	procID string, msgID string, trimJSONDquote bool,
) *AdvancedSyslogFormatter {
	advancedSyslogFormatter := AdvancedSyslogFormatter{
		LevelToSeverity: DefaultLevelToSeverity(),
		Facility:        facility,
		Hostname:        hostname,
		AppName:         rfc5424.AppName(appName),
		ProcID:          rfc5424.ProcID(procID),
		MsgID:           rfc5424.MsgID(msgID),
		TrimJSONDquote:  trimJSONDquote,
	}

	return &advancedSyslogFormatter
}

// Format implements logrus.Formatter interface
func (f *AdvancedSyslogFormatter) Format(entry *log.Entry) ([]byte, error) { //nolint:funlen,gocyclo
	var err error

	detailList := NewJSONDataElement(StructuredIDDetails)

	data := make(log.Fields)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	for _, key := range []string{
		log.FieldKeyLevel, log.FieldKeyTime, log.FieldKeyFunc,
		log.FieldKeyMsg, log.FieldKeyFile, KeyCallStack,
	} {
		prefixFieldClashes(data, key)
	}

	var funcVal, fileVal string
	if entry.HasCaller() {
		funcVal, fileVal = ModuleCallerPrettyfier(entry.Caller)
		detailList.Append(log.FieldKeyFunc, funcVal, f.TrimJSONDquote)
	}
	if errorVal, ok := data[log.ErrorKey]; ok {
		detailList.Append(log.ErrorKey, errorVal, f.TrimJSONDquote)
		delete(data, log.ErrorKey)
	}

	detailKeys := []string{}
	for key := range entry.Data {
		detailKeys = append(detailKeys, key)
	}
	sort.Strings(detailKeys)

	for _, key := range detailKeys {
		detailList.Append(key, entry.Data[key], f.TrimJSONDquote)
	}

	if len(fileVal) > 0 {
		detailList.Append(log.FieldKeyFile, fileVal, f.TrimJSONDquote)
	}

	structuredData := rfc5424.StructuredData{
		detailList,
	}

	msgID := f.MsgID
	if msgID == "" {
		msgID = "DETAILS_MSG"
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

	//textPart := []byte(message.String())
	textPart := []byte(MessageString(message))

	if entry.Context != nil {
		if callStack := entry.Context.Value(ContextLogFieldKey(KeyCallStack)); callStack != nil {
			if callStackLines, ok := callStack.([]string); ok {
				if textPart[len(textPart)-1] != '\n' {
					textPart = append(textPart, '\n')
				}
				textPart = append(textPart, '\t')
				textPart = append(textPart,
					[]byte(strings.Join(callStackLines, "\n\t"))...,
				)
				textPart = append(textPart, '\n')
			}
		}
	}

	return textPart, err
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

// prefixFieldClashes is a copy of Logrus feature
func prefixFieldClashes(data log.Fields, key string) {
	if v, ok := data[key]; ok {
		data["fields."+key] = v
		delete(data, key)
	}
}
