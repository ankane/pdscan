package internal

import (
	"strings"
)

type MatchFinder struct {
	MatchedValues [][]string
	nameIndex     int
}

func NewMatchFinder() MatchFinder {
	return MatchFinder{make([][]string, len(regexRules)+1), len(regexRules)}
}

func (a *MatchFinder) Scan(v string) {
	for i, rule := range regexRules {
		if rule.Regex.MatchString(v) {
			a.MatchedValues[i] = append(a.MatchedValues[i], v)
		}
	}

	tokens := tokenizer.Split(strings.ToLower(v), -1)
	if anyMatches(tokens) {
		a.MatchedValues[a.nameIndex] = append(a.MatchedValues[a.nameIndex], v)
	}
}
