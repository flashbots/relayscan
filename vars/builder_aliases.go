package vars

import (
	"regexp"
	"strings"
)

// BuilderAliases maps builder name to a function that returns if an input string (extra_data) is an alias
var BuilderAliases = map[string]func(string) bool{
	"penguinbuild.org": func(in string) bool {
		return strings.Contains(in, "penguinbuild.org")
	},
	"builder0x69": func(in string) bool {
		return strings.Contains(in, "builder0x69")
	},
	"rsync-builder.xyz": func(in string) bool {
		return strings.Contains(in, "rsync")
	},
	"bob the builder": func(in string) bool {
		match, _ := regexp.MatchString("s[0-9]+e[0-9].*(t|f)", in)
		return match
	},
	"flashbots": func(in string) bool {
		return strings.Contains(in, "Illuminate Dmocr")
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
