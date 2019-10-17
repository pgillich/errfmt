package errfmt

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

type LoggerMock struct {
	*log.Logger
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

func TestMessages(t *testing.T) {
	err := GenerateDeepErrors()

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
