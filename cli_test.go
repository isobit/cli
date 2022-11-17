package cli

import (
	"fmt"
	"net/url"
	"strings"
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
	r := New("test", cmd).
		ParseArgs([]string{
			"--bool",
			"--string", "hello",
			"--int", "42",
		})
	require.NoError(t, r.Err)

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
		Strings           []string `cli:"append"`
	}
	type Subcommand struct {
		Message string
	}

	cmd := &Cmd{
		StringWithDefault: "hello",
		Int64WithDefault:  -123,
	}
	subcmd := &Subcommand{}

	r := New(
		"test", cmd,
		New("subcmd", subcmd),
	).
		ParseArgs([]string{
			"--bool",
			"--string", "hello",
			"--int", "42",
			"--string-pointer", "hello",
			"--blah", "hello",
			"-s", "hello",
			"--int64-pointer", "123",
			"--time", "2022-02-22T22:22:22Z",
			"--duration", "15m",
			"--strings", "a", "--strings", "b",
			"subcmd",
			"--message", "Hello, world!",
		})
	require.NoError(t, r.Err)

	stringPointerValue := "hello"
	int64PointerValue := int64(123)
	timeValue, err := time.Parse(time.RFC3339, "2022-02-22T22:22:22Z")
	require.NoError(t, err)
	durationValue, err := time.ParseDuration("15m")
	require.NoError(t, err)

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
		Strings:           []string{"a", "b"},
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

	r := New("test", cmd).
		ParseArgs([]string{})
	assert.Error(t, r.Err)
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

	r := New("test", cmd).
		ParseArgs([]string{
			"--user", "foo",
			"--punctuation", "!",
		})
	require.NoError(t, r.Err)

	err := r.Run()
	require.NoError(t, err)

	assert.Equal(t, "Hello, foo!", cmd.message)
}

func TestCLIEnvVar(t *testing.T) {
	type Cmd struct {
		Foo string `cli:"env=FOO"`
	}
	cmd := &Cmd{}

	t.Setenv("FOO", "quux")
	r := New("test", cmd).
		ParseArgs([]string{})
	require.NoError(t, r.Err)
	assert.Equal(t, "quux", cmd.Foo)
}

func TestCLIEnvVarPrecedence(t *testing.T) {
	type Cmd struct {
		Foo string `cli:"env=FOO"`
	}
	cmd := &Cmd{}

	t.Setenv("FOO", "quux")
	r := New("test", cmd).
		ParseArgs([]string{
			"--foo", "override",
		})
	require.NoError(t, r.Err)
	assert.Equal(t, "override", cmd.Foo)
}

func TestCLIErrHelp(t *testing.T) {
	r := New("test", &struct{}{}).
		ParseArgs([]string{"--help"})
	assert.Equal(t, r.Err, ErrHelp)
}

func TestCLIPointerWithDefault(t *testing.T) {
	type Cmd struct {
		URL *url.URL
	}
	cmd := &Cmd{
		URL: &url.URL{Scheme: "https", Host: "example.com"},
	}
	r := New("test", cmd).ParseArgs([]string{})
	require.NoError(t, r.Err)

	expected := &Cmd{
		URL: &url.URL{Scheme: "https", Host: "example.com"},
	}
	assert.Equal(t, expected, cmd)
}

func TestCLICustomEnvLookup(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		ErrWriter: b,
		LookupEnv: func(key string) (string, bool, error) {
			return "quux", true, nil
		},
	}

	cmd := &struct {
		Foo string `cli:"env=FOO"`
	}{}
	subcmd := &struct {
		Bar string `cli:"env=BAR"`
	}{}

	r := cli.New(
		"test", cmd,
		cli.New("sub", subcmd),
	).
		ParseArgs([]string{
			"sub",
		})
	require.NoError(t, r.Err)
	assert.Equal(t, "quux", cmd.Foo)
	assert.Equal(t, "quux", subcmd.Bar)
}

func TestCLIEnvLookupError(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		ErrWriter: b,
		LookupEnv: func(key string) (string, bool, error) {
			return "", false, fmt.Errorf("boom")
		},
	}

	cmd := &struct {
		Foo string `cli:"env=FOO"`
	}{}

	r := cli.New("test", cmd).
		ParseArgs([]string{})
	assert.Error(t, r.Err)
}

type testTimeSetter struct {
	t *time.Time
}

func (ts *testTimeSetter) Set(s string) error {
	v, err := time.Parse(time.Kitchen, s)
	if err != nil {
		return err
	}
	*ts.t = v
	return nil
}

func TestCLISetter(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		ErrWriter: b,
		Setter: func(i interface{}) Setter {
			switch v := i.(type) {
			case *time.Time:
				return &testTimeSetter{v}
			default:
				return nil
			}
		},
	}

	cmd := &struct {
		Time time.Time
	}{}

	r := cli.New("test", cmd).
		ParseArgs([]string{
			"--time", "12:30PM",
		})
	require.NoError(t, r.Err)
	assert.Equal(t, time.Time(time.Date(0, time.January, 1, 12, 30, 0, 0, time.UTC)), cmd.Time)
}

func TestCLINilConfig(t *testing.T) {
	r := New("test", nil).
		ParseArgs([]string{})
	require.NoError(t, r.Err)
}

func TestCLIArgsField(t *testing.T) {
	type Cmd struct {
		Bool   bool
		String string
		Int    int
		Args   []string `cli:"args"`
	}
	cmd := &Cmd{}
	r := New("test", cmd).
		ParseArgs([]string{
			"--bool",
			"--string", "hello",
			"--int", "42",
			"hello", "world",
		})
	require.NoError(t, r.Err)

	expected := &Cmd{
		Bool:   true,
		String: "hello",
		Int:    42,
		Args:   []string{"hello", "world"},
	}
	assert.Equal(t, expected, cmd)
}
