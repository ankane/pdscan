package internal

import (
	"regexp"
	"strings"
)

type tableData struct {
	ColumnNames  []string
	ColumnValues [][]string
}

type MatchConfig struct {
	RegexRules     []regexRule
	NameRules      []nameRule
	MultiNameRules []multiNameRule
	TokenRules     []tokenRule
	MinCount       int
}

func NewMatchConfig() MatchConfig {
	return MatchConfig{
		RegexRules:     regexRules,
		NameRules:      nameRules,
		MultiNameRules: multiNameRules,
		TokenRules:     tokenRules,
		MinCount:       1,
	}
}

type MatchFinder struct {
	MatchedValues [][]string
	TokenValues   [][]string
	Count         int
	matchConfig   *MatchConfig
}

var tokenizer = regexp.MustCompile(`\W+`)

func NewMatchFinder(matchConfig *MatchConfig) MatchFinder {
	return MatchFinder{
		make([][]string, len(matchConfig.RegexRules)),
		make([][]string, len(matchConfig.TokenRules)),
		0,
		matchConfig,
	}
}

func (a *MatchFinder) Scan(v string) {
	for i, rule := range a.matchConfig.RegexRules {
		if rule.Regex.MatchString(v) {
			a.MatchedValues[i] = append(a.MatchedValues[i], v)
		}
	}

	if len(a.matchConfig.TokenRules) > 0 {
		tokens := tokenizer.Split(strings.ToLower(v), -1)
		for i, rule := range a.matchConfig.TokenRules {
			if anyMatches(rule, tokens) {
				a.TokenValues[i] = append(a.TokenValues[i], v)
			}
		}
	}
}

func anyMatches(rule tokenRule, values []string) bool {
	for _, value := range values {
		if rule.Tokens.Contains(value) {
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
	a.MatchedValues = make([][]string, len(a.matchConfig.RegexRules))
	a.TokenValues = make([][]string, len(a.matchConfig.TokenRules))
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

		if len(matchedData) >= a.matchConfig.MinCount {
			confidence := "low"
			if rule.Name == "email" || rule.Name == "pattern" || float64(len(matchedData))/float64(count) > 0.5 {
				confidence = "high"
			}

			lineCount := len(matchedData)

			if onlyValues {
				var matchedValues []string
				for _, v := range matchedData {
					v3 := rule.Regex.FindAllString(v, -1)
					matchedValues = append(matchedValues, v3...)
				}
				matchedData = matchedValues
			}

			matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData, LineCount: lineCount, MatchType: "value"})
		}
	}

	for i, rule := range a.matchConfig.TokenRules {
		matchedData := a.TokenValues[i]

		if len(matchedData) >= a.matchConfig.MinCount {
			confidence := "low"
			if float64(len(matchedData))/float64(count) > 0.1 && len(unique(matchedData)) >= 10 {
				confidence = "high"
			}

			lineCount := len(matchedData)

			if onlyValues {
				var matchedValues []string
				for _, v := range matchedData {
					tokens := tokenizer.Split(strings.ToLower(v), -1)
					for _, v2 := range tokens {
						// TODO check all tokens
						if rule.Tokens.Contains(v2) {
							matchedValues = append(matchedValues, v2)
						}
					}
				}
				matchedData = matchedValues
			}

			matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData, LineCount: lineCount, MatchType: "value"})
		}
	}

	return matchList
}

func (a *MatchFinder) CheckTableData(table table, tableData *tableData) []ruleMatch {
	tableMatchList := []ruleMatch{}

	columnNames := tableData.ColumnNames
	columnValues := tableData.ColumnValues

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

			rule := matchNameRule(name, a.matchConfig.NameRules)
			if rule.Name != "" {
				matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: "medium", Identifier: colIdentifier, MatchedData: values, MatchType: "name"})
			}
		}

		tableMatchList = append(tableMatchList, matchList...)
	}

	for _, rule := range a.matchConfig.MultiNameRules {
		var latCol string
		var lonCol string
		for _, col := range columnNames {
			if stringInSlice(col, rule.ColumnNames[0]) {
				latCol = col
			} else if stringInSlice(col, rule.ColumnNames[1]) {
				lonCol = col
			}
		}
		if latCol != "" && lonCol != "" {
			// TODO show data
			tableMatchList = append(tableMatchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: "medium", Identifier: table.displayName() + "." + latCol + "+" + lonCol, MatchType: "name"})
		}
	}

	return tableMatchList
}
