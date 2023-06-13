package common

import (
	"regexp"
	"strings"
)

// BuilderAliases maps builder name to a function that returns if an input string (extra_data) is an alias
var BuilderAliases = map[string]func(string) bool{
	"builder0x69": func(in string) bool {
		return strings.Contains(in, "builder0x69")
	},
	"bob the builder": func(in string) bool {
		match, _ := regexp.MatchString("s[0-9]+e[0-9].*(t|f)", in)
		return match
	},
}

// BuilderNameFromExtraData returns the builder name from the extra_data field
func BuilderNameFromExtraData(extraData string) string {
	for builder, aliasFunc := range BuilderAliases {
		if aliasFunc(extraData) {
			return builder
		}
	}
	return extraData
}
