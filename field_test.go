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

func TestParseUnknownTagError(t *testing.T) {
	cfg := struct {
		Foo string `opts:"asdfasdf"`
	}{}
	_, err := getFieldsFromConfig(&cfg)
	assert.NotNil(t, err)
}
