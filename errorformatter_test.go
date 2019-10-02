package errorformatter

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"emperror.dev/errors"
)

const (
	StaticTimeFormat = "8008-08-08T08:08:08Z"
)

type LoggerMock struct {
	*AdvancedLogger
	outBuf   *bytes.Buffer
	exitCode int
}

func (l *LoggerMock) exit(code int) {
	l.exitCode = code
}

func replaceCallLine(lines string) string {
	linePattern := regexp.MustCompile(`(?m)\.go:\d*`)
	return linePattern.ReplaceAllString(lines, ".go:0")
}

func newWithDetails() error {
	_, err := strconv.Atoi("NO_NUMBER")
	return errors.WrapWithDetails(err, "MESSAGE%0", "K0_1", "V0_1", "K0_2", "V0_2")
}

func makeDeepErrors() error {
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
		"K3\"5", "V3\"doublequote",
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

func TestMessages(t *testing.T) {
	err := makeDeepErrors()

	text := fmt.Sprintf("%s", err)
	assert.Equal(t, `MESSAGE 4: MESSAGE:2: MESSAGE%0: strconv.Atoi: parsing "NO_NUMBER": invalid syntax`, text)
}

func testSortingFuncDecorator(t *testing.T,
	fieldOrder map[string]int,
	expected []string,
	items []string,
) {
	sorter := SortingFuncDecorator(fieldOrder)
	sorter(items)
	assert.Equal(t, expected, items)
}

func TestSortingFuncDecorator(t *testing.T) {
	type testCase struct {
		fieldOrder map[string]int
		expected   []string
		items      []string
	}

	testCases := []testCase{
		{
			map[string]int{},
			[]string{"a", "b", "c", "x", "y", "z"},
			[]string{"a", "b", "c", "x", "y", "z"},
		},
		{
			map[string]int{},
			[]string{"a", "b", "c", "x", "y", "z"},
			[]string{"x", "b", "y", "a", "c", "z"},
		},
		{
			map[string]int{},
			[]string{"a", "b", "c", "x", "y", "z"},
			[]string{"z", "y", "x", "c", "b", "a"},
		},
		{
			map[string]int{"c": 1, "z": 2},
			[]string{"z", "c", "a", "b", "x", "y"},
			[]string{"z", "y", "x", "c", "b", "a"},
		},
		{
			map[string]int{"c": 1, "z": 2, "b": 3, "N": 5},
			[]string{"b", "z", "c", "a", "x", "y"},
			[]string{"z", "y", "x", "c", "b", "a"},
		},
		{
			map[string]int{"c": 2, "z": 2, "b": 2},
			[]string{"b", "c", "z", "a", "x", "y"},
			[]string{"z", "y", "x", "c", "b", "a"},
		},
	}

	for _, test := range testCases {
		testSortingFuncDecorator(t, test.fieldOrder, test.expected, test.items)
	}
}
