package opts

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestParseStructTagInner(t *testing.T) {
	cases := []struct{
		in string
		out map[string]string
	}{
		{
			"",
			map[string]string{},
		},
		{
			"foo",
			map[string]string{
				"foo": "",
			},
		},
		{
			"foo=bar",
			map[string]string{
				"foo": "bar",
			},
		},
		{
			"foo=bar,baz",
			map[string]string{
				"foo": "bar",
				"baz": "",
			},
		},
		{
			"foo=bar,baz=quux",
			map[string]string{
				"foo": "bar",
				"baz": "quux",
			},
		},
		{
			"foo=bar, baz=quux",
			map[string]string{
				"foo": "bar",
				"baz": "quux",
			},
		},
		{
			"foo=bar,baz='quux1,quux2'",
			map[string]string{
				"foo": "bar",
				"baz": "quux1,quux2",
			},
		},
		{
			"foo,bar='one, two',baz=42",
			map[string]string{
				"foo": "",
				"bar": "one, two",
				"baz": "42",
			},
		},
	}

	for _, c := range cases {
		assert.Equal(
			t,
			c.out,
			parseStructTagInner(c.in),
		)
	}
}
