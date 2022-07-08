package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIWritesHelp(t *testing.T) {
	b := &strings.Builder{}
	ctx := Context{
		ErrWriter: b,
	}

	err := ctx.New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		}).
		Run()
	assert.Equal(t, err, ErrHelp)
	assert.NotEmpty(t, b.String())
}
