package cli

import (
	"strings"
)

/*
parseStructTagInner parses the inner part of a struct tag - that is, the
part in double quotes - into a map. The inner string is expected to be a
comma-separated list of key-value pairs. Key-value pairs are expressed as the
key string, followed by "=", followed by the value, which can optionally be
enclosed in single quotes ("'"). For example:

	"foo" -> {"foo": ""}
	"foo=bar" -> {"foo": "bar"}
	"foo='bar'" -> {"foo": "bar"}
*/
func parseStructTagInner(tagInner string) map[string]string {
	ret := map[string]string{}

	key := strings.Builder{}
	val := strings.Builder{}
	inKey := true
	inQuote := false
	for _, c := range tagInner {
		if inKey {
			switch c {
			case ',':
				ret[key.String()] = ""
				key.Reset()
			case '=':
				inKey = false
			case ' ':
				break
			default:
				key.WriteRune(c)
			}
		} else if inQuote {
			switch c {
			case '\'':
				inQuote = false
			default:
				val.WriteRune(c)
			}
		} else {
			switch c {
			case ',':
				ret[key.String()] = val.String()
				key.Reset()
				val.Reset()
				inKey = true
			case '\'':
				inQuote = true
			default:
				val.WriteRune(c)
			}
		}
	}
	if key.Len() > 0 {
		ret[key.String()] = val.String()
	}

	return ret
}
