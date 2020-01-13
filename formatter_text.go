package errfmt

import (
	"sort"

	log "github.com/sirupsen/logrus"
)

/*
NewTextLogger builds a customized Logrus text logger+formatter
	Features:
	* CallStackSkipLast
	* CallStackNewLines and CallStackInFields
	* ModuleCallerPrettyfier
	* PrintStructFieldNames
*/
func NewTextLogger(level log.Level, flags int, callStackSkipLast int,
) *log.Logger {
	logger := log.New()

	logger.Formatter = NewAdvancedTextFormatter(flags, callStackSkipLast)
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

/*
AdvancedTextFormatter is a customized Logrus Text formatter
	Features:
	* ModuleCallerPrettyfier
	* PrintStructFieldNames
	* AdvancedFieldOrder
*/
type AdvancedTextFormatter struct {
	log.TextFormatter
	ConsoleFlags
	AdvancedFormatter
}

// NewAdvancedTextFormatter makes a new AdvancedTextFormatter
func NewAdvancedTextFormatter(flags int, callStackSkipLast int) *AdvancedTextFormatter {
	return &AdvancedTextFormatter{
		TextFormatter: log.TextFormatter{
			CallerPrettyfier: ModuleCallerPrettyfier,
			SortingFunc:      SortingFuncDecorator(AdvancedFieldOrder()),
			DisableColors:    true,
			QuoteEmptyFields: true,
		},
		ConsoleFlags: ConsoleFlags{
			CallStackOnConsole: flags&FlagCallStackOnConsole > 0,
			CallStackSkipLast:  callStackSkipLast,
		},

		AdvancedFormatter: AdvancedFormatter{
			Flags:              flags,
			CallStackSkipLastX: callStackSkipLast,
		},
	}
}

// Format implements logrus.Formatter interface
// calls logrus.TextFormatter.Format()
func (f *AdvancedTextFormatter) Format(entry *log.Entry) ([]byte, error) {
	var consoleCallStackLines []string

	if f.CallStackOnConsole {
		consoleCallStackLines = GetCallStack(entry)
	}

	// consoleCallStackLines cannot be dig anymore
	RenderErrorInEntry(entry)

	textPart, err := f.TextFormatter.Format(entry)

	if len(consoleCallStackLines) > f.CallStackSkipLast {
		textPart = AppendCallStack(textPart, consoleCallStackLines[:len(consoleCallStackLines)-f.CallStackSkipLast])
	}

	return textPart, err
}

// SortingFuncDecorator builds a field sorter with given order (syslog)
// fits to Logrus TextFormatter.SortingFunc
func SortingFuncDecorator(fieldOrder map[string]int) func([]string) {
	return func(keys []string) {
		sorter := EntryFieldSorter{keys, fieldOrder}
		sort.Sort(sorter)
	}
}

// EntryFieldSorter is a type for providing custom field order
type EntryFieldSorter struct {
	items      []string
	fieldOrder map[string]int
}

// Len returns number of elements
func (sorter EntryFieldSorter) Len() int { return len(sorter.items) }

// Swap changes two items
func (sorter EntryFieldSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

// Less returns by given order. If equal, returns alphabetical
func (sorter EntryFieldSorter) Less(i, j int) bool {
	iWeight := sorter.weight(i)
	jWeight := sorter.weight(j)

	if iWeight == jWeight {
		return sorter.items[i] < sorter.items[j]
	}

	return iWeight > jWeight
}

// weight returns the order weight. If order is not specified, returns 0 (alphabetical will be used)
func (sorter EntryFieldSorter) weight(i int) int {
	if weight, ok := sorter.fieldOrder[sorter.items[i]]; ok {
		return weight
	}

	return 0
}

// dropDisabled is in-place drop
/* NOT IMPLEMENTED
func (sorter *EntryFieldSorter) dropDisabled() {
	n := 0
	for _, item := range sorter.items {
		if weight, ok := sorter.fieldOrder[item]; !ok || weight > DisabledFieldWeight {
			sorter.items[n] = item
			n++
		}
	}
	remained := sorter.items[:n]
	sorter.items = remained
}
*/

// AdvancedFieldOrder is the default field order (similar to syslog)
func AdvancedFieldOrder() map[string]int {
	return map[string]int{
		log.FieldKeyLevel:       100, // first
		log.FieldKeyTime:        90,
		KeyHandlerFunc:          81,
		log.FieldKeyFunc:        80,
		log.ErrorKey:            70,
		log.FieldKeyMsg:         60,
		log.FieldKeyLogrusError: 50,
		log.FieldKeyFile:        40,
		KeyCallStack:            -10, // after normal fields

		// value DisabledFieldWeight drops the field
		//KeyCallStackHidden: DisabledFieldWeight,
	}
}
