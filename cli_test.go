package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIBasic(t *testing.T) {
	type App struct {
		Bool   bool
		String string
		Int    int
	}
	app := &App{}
	po := New("test", app).
		ParseArgs([]string{
			"test",
			"--bool",
			"--string", "hello",
			"--int", "42",
		})
	require.Nil(t, po.Err)

	expected := &App{
		Bool:   true,
		String: "hello",
		Int:    42,
	}
	assert.Equal(t, expected, app)
}

func TestCLIKitchenSink(t *testing.T) {
	type App struct {
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
		hidden            int
		RepeatedString    []string `cli:"repeatable"`
	}
	type Subcommand struct {
		Message string
	}

	app := &App{
		StringWithDefault: "hello",
		Int64WithDefault:  -123,
	}
	subcmd := &Subcommand{}

	po := New("test", app).
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

	appExpected := &App{
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
	assert.Equal(t, appExpected, app)
	assert.Equal(t, subcmdExpected, subcmd)
}

func TestCLIRequired(t *testing.T) {
	type App struct {
		Foo string `cli:"required"`
	}
	app := &App{}

	po := New("test", app).
		ParseArgs([]string{
			"test",
		})
	assert.NotNil(t, po.Err)
}

type cliRunTestApp struct {
	Punctuation string
	User        string
	fmtString   string
	message     string
}

func (app *cliRunTestApp) Before() error {
	app.fmtString = "Hello, %s" + app.Punctuation
	return nil
}

func (app *cliRunTestApp) Run() error {
	app.message = fmt.Sprintf(app.fmtString, app.User)
	return nil
}

func TestCLIRun(t *testing.T) {
	app := &cliRunTestApp{}

	po := New("test", app).
		ParseArgs([]string{
			"test",
			"--user", "foo",
			"--punctuation", "!",
		})
	require.Nil(t, po.Err)

	err := po.Run()
	require.Nil(t, err)

	assert.Equal(t, "Hello, foo!", app.message)
}

func TestCLIEnvVar(t *testing.T) {
	type App struct {
		Foo string `cli:"env=FOO"`
	}
	app := &App{}

	t.Setenv("FOO", "quux")
	po := New("test", app).
		ParseArgs([]string{
			"test",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, "quux", app.Foo)
}

func TestCLIEnvVarPrecedence(t *testing.T) {
	type App struct {
		Foo string `cli:"env=FOO"`
	}
	app := &App{}

	t.Setenv("FOO", "quux")
	po := New("test", app).
		ParseArgs([]string{
			"test", "--foo", "override",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, "override", app.Foo)
}

func TestCLIShortName(t *testing.T) {
	type App struct{}
	type Subcmd struct{}

	po := New("test", &App{}).
		AddCommand(New("subcmd", &Subcmd{}).SetShortName("s")).
		ParseArgs([]string{
			"test", "s",
		})
	assert.Nil(t, po.Err)
}

func TestCLIErrHelp(t *testing.T) {
	po := New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		})
	assert.Equal(t, po.Err, ErrHelp)
}