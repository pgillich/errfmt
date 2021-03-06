package errfmt

import (
	"sort"

	log "github.com/sirupsen/logrus"
)

/*
NewTextLogger builds a customized Logrus JSON logger+formatter
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
		AdvancedFormatter: AdvancedFormatter{
			Flags:             flags,
			CallStackSkipLast: callStackSkipLast,
		},
	}
}

// Format implements logrus.Formatter interface
// nolint:gocyclo,funlen
func (f *AdvancedTextFormatter) Format(entry *log.Entry) ([]byte, error) {
	entry.Data = f.MergeDetailsToFields(entry)
	callStackLines := f.GetCallStack(entry)
	if (f.Flags & FlagCallStackInFields) > 0 {
		entry.Data[KeyCallStack] = callStackLines
	}

	f.RenderFieldValues(entry.Data)

	textPart, err := f.TextFormatter.Format(entry)

	if (f.Flags & FlagCallStackOnConsole) > 0 {
		textPart = f.AppendCallStack(textPart, callStackLines)
	}

	return textPart, err
}

// SortingFuncDecorator builds a field sorter with given order (syslog)
// fits to Logrus TextFormatter.SortingFunc
func SortingFuncDecorator(fieldOrder map[string]int) func([]string) {
	return func(keys []string) {
		sorter := EntryFieldSorter{keys, fieldOrder}
		sorter.dropDisabled()
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
func (sorter EntryFieldSorter) dropDisabled() {
	n := 0
	for _, item := range sorter.items {
		if weight, ok := sorter.fieldOrder[item]; !ok || weight > DisabledFieldWeight {
			sorter.items[n] = item
			n++
		}
	}
	sorter.items = sorter.items[:n]
}

// AdvancedFieldOrder is the default field order (similar to syslog)
func AdvancedFieldOrder() map[string]int {
	return map[string]int{
		log.FieldKeyLevel:       100, // first
		log.FieldKeyTime:        90,
		log.FieldKeyFunc:        80,
		log.ErrorKey:            70,
		log.FieldKeyMsg:         60,
		log.FieldKeyLogrusError: 50,
		log.FieldKeyFile:        40,
		KeyCallStack:            -10, // after normal fields
	}
}
