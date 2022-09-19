package internal

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type ruleMatch struct {
	RuleName    string
	DisplayName string
	Confidence  string
	Identifier  string
	MatchedData []string
	MatchType   string
	LineCount   int
}

type matchInfo struct {
	ruleMatch
	RowStr string
	Values []string
}

func unique(arr []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range arr {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func matchNameRule(name string, rules []nameRule) nameRule {
	for _, rule := range rules {
		if stringInSlice(name, rule.ColumnNames) {
			return rule
		}
	}
	return nameRule{}
}

var space = regexp.MustCompile(`\s+`)
var urlPassword = regexp.MustCompile(`((\/\/|%2F%2F)\S+(:|%3A))\S+(@|%40)`)

func pluralize(count int, singular string) string {
	if count != 1 {
		if singular == "index" {
			singular = "indices"
		} else if strings.HasSuffix(singular, "ch") {
			singular = singular + "es"
		} else {
			singular = singular + "s"
		}
	}
	return fmt.Sprintf("%d %s", count, singular)
}

func printMatchList(formatter Formatter, matchList []ruleMatch, showData bool, showAll bool, rowStr string) error {
	for _, match := range matchList {
		if showAll || match.Confidence != "low" {
			var values []string
			if showData {
				v := unique(match.MatchedData)
				if len(v) > 0 {
					if len(v) > 50 {
						v = v[0:50]
					}

					for i, v2 := range v {
						v[i] = space.ReplaceAllString(v2, " ")
					}
					sort.Strings(v)
				}
				values = v
			}

			err := formatter.PrintMatch(os.Stdout, matchInfo{match, rowStr, values})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func showLowConfidenceMatchHelp(matchList []ruleMatch) {
	lowConfidenceMatches := []ruleMatch{}
	for _, match := range matchList {
		if match.Confidence == "low" {
			lowConfidenceMatches = append(lowConfidenceMatches, match)
		}
	}
	if len(lowConfidenceMatches) > 0 {
		fmt.Fprintln(os.Stderr, "Also found "+pluralize(len(lowConfidenceMatches), "low confidence match")+". Use --show-all to view them")
	}
}
