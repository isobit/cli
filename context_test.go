package cli

import (
	"fmt"
	"strings"
	"testing"

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
