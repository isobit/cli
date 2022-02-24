package opts

import (
	// "regexp"
	"strings"
)

// var structTagRegex = regexp.MustCompile(`(\w+):"([^"]+)"`)
// var structTagRegex = regexp.MustCompile(`([^ ":]+):"([^"]+)"`)

// func parseStructTag(tag string) map[string](map[string]string) {
// 	ret := map[string]map[string]string{}
// 	matches := structTagRegex.FindAllStringSubmatch(tag, -1)
// 	for _, match := range matches {
// 		ret[match[1]] = parseStructTagInner(match[2])
// 	}
// 	return ret
// }

func parseStructTagInner(tagInner string) map[string]string {
	ret := map[string]string{}
	items := strings.Split(tagInner, ",")

	for _, item := range items {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) > 1 {
			ret[kv[0]] = kv[1]
		} else {
			ret[kv[0]] = ""
		}

	}
	return ret
}
