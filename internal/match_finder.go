package internal

import (
	"regexp"
	"strings"

	"github.com/deckarep/golang-set"
)

type MatchConfig struct {
	RegexRules   []regexRule
	NameRules    []nameRule
	LastNamesSet mapset.Set
}

func NewMatchConfig() MatchConfig {
	return MatchConfig{
		RegexRules:   regexRules,
		NameRules:    nameRules,
		LastNamesSet: lastNamesSet,
	}
}

type MatchFinder struct {
	MatchedValues [][]string
	Count         int
	nameIndex     int
	matchConfig   *MatchConfig
}

var tokenizer = regexp.MustCompile(`\W+`)

func NewMatchFinder(matchConfig *MatchConfig) MatchFinder {
	regexRules := matchConfig.RegexRules
	nameIndex := len(regexRules)
	return MatchFinder{make([][]string, nameIndex+1), 0, nameIndex, matchConfig}
}

func (a *MatchFinder) Scan(v string) {
	for i, rule := range a.matchConfig.RegexRules {
		if rule.Regex.MatchString(v) {
			a.MatchedValues[i] = append(a.MatchedValues[i], v)
		}
	}

	tokens := tokenizer.Split(strings.ToLower(v), -1)
	if a.anyMatches(tokens) {
		a.MatchedValues[a.nameIndex] = append(a.MatchedValues[a.nameIndex], v)
	}
}

func (a *MatchFinder) anyMatches(values []string) bool {
	for _, value := range values {
		if a.matchConfig.LastNamesSet.Contains(value) {
			return true
		}
	}
	return false
}

func (a *MatchFinder) ScanValues(values []string) {
	for _, v := range values {
		a.Scan(v)
	}
	a.Count += len(values)
}

func (a *MatchFinder) Clear() {
	a.MatchedValues = make([][]string, a.nameIndex+1)
	a.Count = 0
}

func (a *MatchFinder) CheckMatches(colIdentifier string, onlyValues bool) []ruleMatch {
	matchList := []ruleMatch{}

	matchedValues := a.MatchedValues
	count := a.Count

	for i, rule := range a.matchConfig.RegexRules {
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

func (a *MatchFinder) CheckTableData(table table, columnNames []string, columnValues [][]string) []ruleMatch {
	tableMatchList := []ruleMatch{}

	for i, col := range columnNames {
		// check values
		values := columnValues[i]

		var colIdentifier string
		if table.displayName() == "" {
			colIdentifier = col
		} else {
			colIdentifier = table.displayName() + "." + col
		}

		a.Clear()
		a.ScanValues(values)
		matchList := a.CheckMatches(colIdentifier, false)

		// only check name if no matches
		if len(matchList) == 0 {
			name := strings.Replace(strings.ToLower(col), "_", "", -1)

			// check last part for nested data
			parts := strings.Split(name, ".")
			name = parts[len(parts)-1]

			rule := matchNameRule(name, nameRules)
			if rule.Name != "" {
				matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: "medium", Identifier: colIdentifier, MatchedData: values, MatchType: "name"})
			}
		}

		tableMatchList = append(tableMatchList, matchList...)
	}

	// check for location data
	var latCol string
	var lonCol string
	for _, col := range columnNames {
		if stringInSlice(col, []string{"latitude", "lat"}) {
			latCol = col
		} else if stringInSlice(col, []string{"longitude", "lon", "lng"}) {
			lonCol = col
		}
	}
	if latCol != "" && lonCol != "" {
		// TODO show data
		tableMatchList = append(tableMatchList, ruleMatch{RuleName: "location", DisplayName: "location data", Confidence: "medium", Identifier: table.displayName() + "." + latCol + "+" + lonCol, MatchType: "name"})
	}

	return tableMatchList
}
