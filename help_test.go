package opts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptsWritesHelp(t *testing.T) {
	oldErrWriter := errWriter
	defer func() {
		errWriter = oldErrWriter
	}()
	b := &strings.Builder{}
	errWriter = b

	err := New("test", &struct{}{}).
		ParseArgs([]string{
			"test", "--help",
		}).
		Run()
	assert.Equal(t, err, ErrHelp)
	assert.NotEmpty(t, b.String())
}
