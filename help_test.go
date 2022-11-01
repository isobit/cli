package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIWritesHelp(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		HelpWriter: b,
	}

	err := cli.New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		}).
		Run()
	assert.Equal(t, err, ErrHelp)
	assert.NotEmpty(t, b.String())
}

func TestCLIInvalidUsageWritesHelp(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		HelpWriter: b,
	}

	err := cli.New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--undefined",
		}).
		Run()
	assert.Error(t, err)
	assert.NotEmpty(t, b.String())
}

type helpTestCommand struct {
	beforeErr error
	runErr    error
}

func (cmd *helpTestCommand) Before() error {
	return cmd.beforeErr
}

func (cmd *helpTestCommand) Run() error {
	return cmd.runErr
}

func TestCLIUsageErrors(t *testing.T) {
	boom := fmt.Errorf("boom!")
	testCases := []struct {
		beforeErr       error
		runErr          error
		shouldPrintHelp bool
	}{
		{
			beforeErr:       nil,
			runErr:          nil,
			shouldPrintHelp: false,
		},
		{
			beforeErr:       boom,
			runErr:          nil,
			shouldPrintHelp: false,
		},
		{
			beforeErr:       nil,
			runErr:          boom,
			shouldPrintHelp: false,
		},
		{
			beforeErr:       UsageError(boom),
			runErr:          nil,
			shouldPrintHelp: true,
		},
		{
			beforeErr:       nil,
			runErr:          UsageError(boom),
			shouldPrintHelp: true,
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b := &strings.Builder{}
			cli := CLI{
				HelpWriter: b,
			}
			cmd := &helpTestCommand{
				beforeErr: testCase.beforeErr,
				runErr:    testCase.runErr,
			}
			err := cli.New("test", cmd).
				ParseArgs([]string{"test"}).
				Run()
			if testCase.beforeErr != nil {
				assert.Equal(t, testCase.beforeErr, err)
			}
			if testCase.runErr != nil {
				assert.Equal(t, testCase.runErr, err)
			}
			if testCase.shouldPrintHelp {
				assert.NotEmpty(t, b.String())
			} else {
				assert.Empty(t, b.String())
			}
		})
	}
}
