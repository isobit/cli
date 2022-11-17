package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldIgnoreMinusTag(t *testing.T) {
	type Cfg struct {
		Ignored string `cli:"-"`
	}
	fields, _, err := defaultCLI.getFieldsFromConfig(&Cfg{})
	require.NoError(t, err)
	assert.Len(t, fields, 0)
}

func TestFieldUnknownTagError(t *testing.T) {
	type Cfg struct {
		Foo string `cli:"asdfasdf"`
	}
	_, _, err := defaultCLI.getFieldsFromConfig(&Cfg{})
	assert.Error(t, err)
}

func TestFieldEmbedded(t *testing.T) {
	type EmbeddedCfg struct {
		Bar string
	}
	type Cfg struct {
		Foo string
		EmbeddedCfg
	}
	fields, _, err := defaultCLI.getFieldsFromConfig(&Cfg{})
	require.NoError(t, err)
	assert.Len(t, fields, 2)
	assert.Equal(t, "foo", fields[0].Name)
	assert.Equal(t, "bar", fields[1].Name)
}

func TestFieldAppend(t *testing.T) {
	getFieldSet := func(t *testing.T, cfg interface{}) func(s string) {
		fields, _, err := defaultCLI.getFieldsFromConfig(cfg)
		require.NoError(t, err)
		require.Len(t, fields, 1)
		flag := fields[0].flagValue
		return func(s string) {
			err := flag.Set(s)
			require.NoError(t, err)
		}
	}
	t.Run("[]string", func(t *testing.T) {
		cfg := struct {
			Vars []string `cli:"append,short=v"`
		}{}
		set := getFieldSet(t, &cfg)
		set("aaa")
		set("bbb")
		set("ccc")
		assert.Equal(t, []string{"aaa", "bbb", "ccc"}, cfg.Vars)
	})
	t.Run("[]*string", func(t *testing.T) {
		cfg := struct {
			Vars []*string `cli:"append,short=v"`
		}{}
		set := getFieldSet(t, &cfg)
		set("aaa")
		set("bbb")
		set("ccc")
		s := func(v string) *string { return &v }
		assert.EqualValues(t, []*string{s("aaa"), s("bbb"), s("ccc")}, cfg.Vars)
	})
	t.Run("[]int", func(t *testing.T) {
		cfg := struct {
			Vars []int `cli:"append,short=v"`
		}{}
		set := getFieldSet(t, &cfg)
		set("1")
		set("2")
		set("3")
		assert.Equal(t, []int{1, 2, 3}, cfg.Vars)
	})
	t.Run("[]*int", func(t *testing.T) {
		cfg := struct {
			Vars []*int `cli:"append,short=v"`
		}{}
		set := getFieldSet(t, &cfg)
		set("1")
		set("2")
		set("3")
		i := func(v int) *int { return &v }
		assert.EqualValues(t, []*int{i(1), i(2), i(3)}, cfg.Vars)
	})
}
