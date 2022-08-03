package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIWritesHelp(t *testing.T) {
	b := &strings.Builder{}
	cli := CLI{
		ErrWriter: b,
	}

	err := cli.New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		}).
		Run()
	assert.Equal(t, err, ErrHelp)
	assert.NotEmpty(t, b.String())
}
