package internal

import (
	"strings"
)

type MatchFinder struct {
	MatchedValues [][]string
	Count         int
	nameIndex     int
}

func NewMatchFinder() MatchFinder {
	nameIndex := len(regexRules)
	return MatchFinder{make([][]string, nameIndex+1), 0, nameIndex}
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

func (a *MatchFinder) Clear() {
	a.MatchedValues = make([][]string, a.nameIndex+1)
	a.Count = 0
}

func (a *MatchFinder) CheckMatches(colIdentifier string, onlyValues bool) []ruleMatch {
	matchList := []ruleMatch{}

	matchedValues := a.MatchedValues
	count := a.Count

	for i, rule := range regexRules {
		matchedData := matchedValues[i]

		if rule.Name == "email" {
			// filter out false positives with URL credentials
			newMatchedData := matchedData
			matchedData = []string{}
			for _, v := range newMatchedData {
				// replace urls and check for email match again
				v2 := urlPassword.ReplaceAllString(v, "[FILTERED]")
				if rule.Regex.MatchString(v2) {
					matchedData = append(matchedData, v)
				}
			}
		}

		if len(matchedData) > 0 {
			confidence := "low"
			if rule.Name == "email" || float64(len(matchedData))/float64(count) > 0.5 {
				confidence = "high"
			}

			if onlyValues {
				var matchedValues []string
				for _, v := range matchedData {
					v3 := rule.Regex.FindAllString(v, -1)
					matchedValues = append(matchedValues, v3...)
				}
				matchedData = matchedValues
			}

			matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData})
		}
	}

	// find names
	nameIndex := a.nameIndex
	matchedData := matchedValues[nameIndex]

	if len(matchedData) > 0 {
		confidence := "low"
		if float64(len(matchedData))/float64(count) > 0.1 && len(unique(matchedData)) >= 10 {
			confidence = "high"
		}

		if onlyValues {
			var matchedValues []string
			for _, v := range matchedData {
				tokens := tokenizer.Split(strings.ToLower(v), -1)
				for _, v2 := range tokens {
					if lastNamesSet.Contains(v2) {
						matchedValues = append(matchedValues, v2)
					}
				}
			}
			matchedData = matchedValues
		}

		matchList = append(matchList, ruleMatch{RuleName: "last_name", DisplayName: "last names", Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData})
	}

	return matchList
}
