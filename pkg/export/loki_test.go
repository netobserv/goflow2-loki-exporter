package export

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/mock"
)

type fakeEmitter struct {
	mock.Mock
}

func (f *fakeEmitter) Handle(labels model.LabelSet, timestamp time.Time, record string) error {
	a := f.Mock.Called(labels, timestamp, record)
	return a.Error(0)
}

func TestLoki_Process(t *testing.T) {
	// GIVEN a Loki exporter
	fe := fakeEmitter{}
	fe.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cfg, err := ReadConfig(strings.NewReader(`
timestampLabel: ts
ignoreList:
  - ignored
staticLabels:
  static: label
labels:
  - foo
  - bar
printInput: true
printOutput: true
`))
	require.NoError(t, err)
	loki, err := NewLoki(cfg)
	require.NoError(t, err)
	loki.emitter = &fe

	// WHEN it processes input records
	require.NoError(t, loki.Process(strings.NewReader(`
{"ts":123456,"ignored":"ignored!","foo":"fooLabel","bar":"barLabel","value":1234}
{"ts":124567,"ignored":"ignored!","foo":"fooLabel2","bar":"barLabel2","value":5678,"other":"val"}
`)))

	// THEN it forwards the records extracting the timestamp and labels from the configuration
	fe.AssertCalled(t, "Handle", model.LabelSet{
		"app":    "goflow2",
		"bar":    "barLabel",
		"foo":    "fooLabel",
		"static": "label",
	}, time.Unix(123456, 0), `{"ts":123456,"value":1234}`)

	fe.AssertCalled(t, "Handle", model.LabelSet{
		"app":    "goflow2",
		"bar":    "barLabel2",
		"foo":    "fooLabel2",
		"static": "label",
	}, time.Unix(124567, 0), `{"other":"val","ts":124567,"value":5678}`)
}

func TestTimestampScale(t *testing.T) {
	// verifies that the unix residual time (below 1-second precision) is properly
	// incorporated into the timestamp whichever scale it is
	for _, testCase := range []struct {
		unit     string
		expected time.Time
	}{
		{unit: "1m", expected: time.Unix(123456789*60, 0)},
		{unit: "1s", expected: time.Unix(123456789, 0)},
		{unit: "100ms", expected: time.Unix(12345678, 900000000)},
		{unit: "1ms", expected: time.Unix(123456, 789000000)},
	} {
		t.Run(fmt.Sprintf("unit %v", testCase.unit), func(t *testing.T) {
			fe := fakeEmitter{}
			fe.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			cfg, err := ReadConfig(strings.NewReader(
				fmt.Sprintf("timestampScale: %s", testCase.unit)))
			require.NoError(t, err)
			loki, err := NewLoki(cfg)
			require.NoError(t, err)
			loki.emitter = &fe

			require.NoError(t, loki.Process(strings.NewReader(`{"TimeReceived":123456789}`+"\n")))
			fe.AssertCalled(t, "Handle", model.LabelSet{"app": "goflow2"},
				testCase.expected, `{"TimeReceived":123456789}`)
		})
	}
}

// Tests those cases where the timestamp can't be extracted and reports the current time
func TestTimestampExtraction_LocalTime(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		tsLabel model.LabelName
		input   string
	}{
		{name: "undefined ts label", tsLabel: "", input: `{"ts":444}`},
		{name: "non-existing ts entry", tsLabel: "asdfasdf", input: `{"ts":444}`},
		{name: "non-numeric ts value", tsLabel: "ts", input: `{"ts":"string value"}`},
		{name: "zero ts value", tsLabel: "ts", input: `{"ts":0}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fe := fakeEmitter{}
			fe.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			cfg := DefaultConfig()
			cfg.TimestampLabel = testCase.tsLabel
			loki, err := NewLoki(cfg)
			require.NoError(t, err)
			loki.emitter = &fe
			loki.timeNow = func() time.Time {
				return time.Unix(12345678, 0)
			}
			require.NoError(t, loki.Process(strings.NewReader(testCase.input)))
			fe.AssertCalled(t, "Handle", model.LabelSet{"app": "goflow2"},
				time.Unix(12345678, 0), testCase.input)
		})
	}
}

// Tests that labels are sanitized before being sent to loki.
// Labels that are invalid even if sanitized are ignored
func TestSanitizedLabels(t *testing.T) {
	fe := fakeEmitter{}
	fe.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cfg, err := ReadConfig(strings.NewReader(`
labels:
  - "fo.o"
  - "ba-r"
  - "ba/z"
  - "ignored?"
`))
	require.NoError(t, err)
	loki, err := NewLoki(cfg)
	require.NoError(t, err)
	loki.emitter = &fe

	require.NoError(t, loki.Process(strings.NewReader(`
{"ba/z":"isBaz","fo.o":"isFoo","ba-r":"isBar","ignored?":"yes!"}
`)))

	fe.AssertCalled(t, "Handle", model.LabelSet{
		"app":  "goflow2",
		"ba_r": "isBar",
		"fo_o": "isFoo",
		"ba_z": "isBaz",
	}, mock.Anything, mock.Anything)
}
