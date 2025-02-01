package utils

import (
	"log"
	"regexp"
	"rss-reader/globals"
	"strings"
)

func MatchAllowList(str string) bool {
	return MatchStr(str, globals.MatchList)
}

func MatchDenyList(str string) bool {
	return MatchStr(str, globals.DenyMatchList)
}

func MatchStr(str string, matchList []string) bool {
	strFinal := strings.TrimSpace(str)
	for _, pattern := range matchList {
		pattern = "(?i)" + pattern
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("⚠️ Invalid regular expression: %s, error: %v", pattern, err)
			continue
		}

		if re.MatchString(strFinal) {
			return true
		}
	}
	return false
}
