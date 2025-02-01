package utils

import (
	"rss-reader/globals"
	"strings"
)

func MatchStr(str string, callback func(string)) {
	strFinal := strings.ToLower(strings.TrimSpace(str))
	for _, v := range globals.MatchList {
		v = strings.ToLower(strings.TrimSpace(v))
		if strings.Contains(strFinal, v) {
			callback(str)
			return
		}
	}
}
