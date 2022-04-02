package opts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalBase64String(t *testing.T) {
	input := []byte("SGVsbG8sIHdvcmxkIQ==")
	var output Base64String
	err := (&output).UnmarshalText(input)
	require.Nil(t, err)
	assert.Equal(t, "Hello, world!", string(output))
}
