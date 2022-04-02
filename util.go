package opts

import (
	"encoding/base64"
)

// Base64String is a byte slice that can be unmarshaled from a standard (RFC
// 4648) base64-encoded string.
type Base64String []byte

func (b *Base64String) UnmarshalText(src []byte) error {
	enc := base64.StdEncoding
	dbuf := make([]byte, enc.DecodedLen(len(src)))
	n, err := enc.Decode(dbuf, src)
	if err != nil {
		return err
	}
	*b = dbuf[:n]
	return nil
}
