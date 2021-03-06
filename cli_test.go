package cli

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIBasic(t *testing.T) {
	type Cmd struct {
		Bool   bool
		String string
		Int    int
	}
	cmd := &Cmd{}
	po := New("test", cmd).
		ParseArgs([]string{
			"test",
			"--bool",
			"--string", "hello",
			"--int", "42",
		})
	require.Nil(t, po.Err)

	expected := &Cmd{
		Bool:   true,
		String: "hello",
		Int:    42,
	}
	assert.Equal(t, expected, cmd)
}

func TestCLIKitchenSink(t *testing.T) {
	type Cmd struct {
		Bool              bool
		String            string
		Int               int
		StringPointer     *string
		StringZeroValue   string
		StringWithDefault string
		StringWithName    string `cli:"name=blah"`
		StringWithShort   string `cli:"short=s"`
		Int64Pointer      *int64
		Int64WithDefault  int64
		Time              time.Time
		Duration          time.Duration
		unexportedInt     int
		RepeatedString    []string `cli:"repeatable"`
	}
	type Subcommand struct {
		Message string
	}

	cmd := &Cmd{
		StringWithDefault: "hello",
		Int64WithDefault:  -123,
	}
	subcmd := &Subcommand{}

	po := New("test", cmd).
		AddCommand(New("subcmd", subcmd)).
		ParseArgs([]string{
			"test",
			"--bool",
			"--string", "hello",
			"--int", "42",
			"--string-pointer", "hello",
			"--blah", "hello",
			"-s", "hello",
			"--int64-pointer", "123",
			"--time", "2022-02-22T22:22:22Z",
			"--duration", "15m",
			"--repeated-string", "a", "--repeated-string", "b",
			"subcmd",
			"--message", "Hello, world!",
		})
	require.Nil(t, po.Err)

	stringPointerValue := "hello"
	int64PointerValue := int64(123)
	timeValue, err := time.Parse(time.RFC3339, "2022-02-22T22:22:22Z")
	require.Nil(t, err)
	durationValue, err := time.ParseDuration("15m")
	require.Nil(t, err)

	cmdExpected := &Cmd{
		Bool:              true,
		String:            "hello",
		Int:               42,
		StringPointer:     &stringPointerValue,
		StringZeroValue:   "",
		StringWithDefault: "hello",
		StringWithName:    "hello",
		StringWithShort:   "hello",
		Int64Pointer:      &int64PointerValue,
		Int64WithDefault:  -123,
		Time:              timeValue,
		Duration:          durationValue,
		RepeatedString:    []string{"a", "b"},
	}
	subcmdExpected := &Subcommand{
		Message: "Hello, world!",
	}
	assert.Equal(t, cmdExpected, cmd)
	assert.Equal(t, subcmdExpected, subcmd)
}

func TestCLIRequired(t *testing.T) {
	type Cmd struct {
		Foo string `cli:"required"`
	}
	cmd := &Cmd{}

	po := New("test", cmd).
		ParseArgs([]string{
			"test",
		})
	assert.NotNil(t, po.Err)
}

type cliRunTestCmd struct {
	Punctuation string
	User        string
	fmtString   string
	message     string
}

func (cmd *cliRunTestCmd) Before() error {
	cmd.fmtString = "Hello, %s" + cmd.Punctuation
	return nil
}

func (cmd *cliRunTestCmd) Run() error {
	cmd.message = fmt.Sprintf(cmd.fmtString, cmd.User)
	return nil
}

func TestCLIRun(t *testing.T) {
	cmd := &cliRunTestCmd{}

	po := New("test", cmd).
		ParseArgs([]string{
			"test",
			"--user", "foo",
			"--punctuation", "!",
		})
	require.Nil(t, po.Err)

	err := po.Run()
	require.Nil(t, err)

	assert.Equal(t, "Hello, foo!", cmd.message)
}

func TestCLIEnvVar(t *testing.T) {
	type Cmd struct {
		Foo string `cli:"env=FOO"`
	}
	cmd := &Cmd{}

	t.Setenv("FOO", "quux")
	po := New("test", cmd).
		ParseArgs([]string{
			"test",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, "quux", cmd.Foo)
}

func TestCLIEnvVarPrecedence(t *testing.T) {
	type Cmd struct {
		Foo string `cli:"env=FOO"`
	}
	cmd := &Cmd{}

	t.Setenv("FOO", "quux")
	po := New("test", cmd).
		ParseArgs([]string{
			"test", "--foo", "override",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, "override", cmd.Foo)
}

func TestCLIErrHelp(t *testing.T) {
	po := New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		})
	assert.Equal(t, po.Err, ErrHelp)
}

func TestCLIPointerWithDefault(t *testing.T) {
	type Cmd struct {
		URL *url.URL
	}
	cmd := &Cmd{
		URL: &url.URL{Scheme: "https", Host: "example.com"},
	}
	po := New("test", cmd).ParseArgs([]string{"test"})
	require.Nil(t, po.Err)

	expected := &Cmd{
		URL: &url.URL{Scheme: "https", Host: "example.com"},
	}
	assert.Equal(t, expected, cmd)
}
