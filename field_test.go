package opts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldIgnoreMinusTag(t *testing.T) {
	cfg := struct {
		Hidden string `opts:"-"`
	}{}
	fields, err := getFieldsFromConfig(&cfg)
	require.Nil(t, err)
	assert.Len(t, fields, 0)
}

func TestFieldRepeatable(t *testing.T) {
	getFieldSet := func(cfg interface{}) func(s string) {
		fields, err := getFieldsFromConfig(cfg)
		require.Nil(t, err)
		require.Len(t, fields, 1)
		flag := fields[0].flagValue
		return func(s string) {
			err := flag.Set(s)
			require.Nil(t, err)
		}
	}
	t.Run("[]string", func(t *testing.T) {
		cfg := struct {
			Vars []string `opts:"repeatable,short=v"`
		}{}
		set := getFieldSet(&cfg)
		set("aaa")
		set("bbb")
		set("ccc")
		assert.Equal(t, []string{"aaa", "bbb", "ccc"}, cfg.Vars)
	})
	t.Run("[]*string", func(t *testing.T) {
		cfg := struct {
			Vars []*string `opts:"repeatable,short=v"`
		}{}
		set := getFieldSet(&cfg)
		set("aaa")
		set("bbb")
		set("ccc")
		s := func(v string) *string { return &v }
		assert.EqualValues(t, []*string{s("aaa"), s("bbb"), s("ccc")}, cfg.Vars)
	})
	t.Run("[]int", func(t *testing.T) {
		cfg := struct {
			Vars []int `opts:"repeatable,short=v"`
		}{}
		set := getFieldSet(&cfg)
		set("1")
		set("2")
		set("3")
		assert.Equal(t, []int{1, 2, 3}, cfg.Vars)
	})
	t.Run("[]*int", func(t *testing.T) {
		cfg := struct {
			Vars []*int `opts:"repeatable,short=v"`
		}{}
		set := getFieldSet(&cfg)
		set("1")
		set("2")
		set("3")
		i := func(v int) *int { return &v }
		assert.EqualValues(t, []*int{i(1), i(2), i(3)}, cfg.Vars)
	})
}
