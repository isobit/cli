package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEnvLookup(t *testing.T) {
	b := &strings.Builder{}
	ctx := Context{
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

	po := ctx.New("test", cmd).
		AddCommand(ctx.New("sub", subcmd)).
		ParseArgs([]string{
			"test", "sub",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, "quux", cmd.Foo)
	assert.Equal(t, "quux", subcmd.Bar)
}

func TestContextEnvLookupError(t *testing.T) {
	b := &strings.Builder{}
	ctx := Context{
		ErrWriter: b,
		LookupEnv: func(key string) (string, bool, error) {
			return "", false, fmt.Errorf("boom")
		},
	}

	cmd := &struct {
		Foo string `cli:"env=FOO"`
	}{}

	po := ctx.New("test", cmd).
		ParseArgs([]string{
			"test",
		})
	assert.NotNil(t, po.Err)
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

func TestContextSetter(t *testing.T) {
	b := &strings.Builder{}
	ctx := Context{
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

	po := ctx.New("test", cmd).
		ParseArgs([]string{
			"test", "--time", "12:30PM",
		})
	require.Nil(t, po.Err)
	assert.Equal(t, time.Time(time.Date(0, time.January, 1, 12, 30, 0, 0, time.UTC)), cmd.Time)
}
