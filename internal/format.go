package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// Format defines the interface used to deliver results to the end user.
type Formatter interface {
	// PrintMatches formats and prints the matches to `writer`.
	PrintMatches(
		writer io.Writer, matches []ruleMatch,
		showData bool, showAll bool, rowKind string,
	) error
}

// Formatters holds available formatters
var Formatters = map[string]Formatter{
	"text": TextFormatter{},
	"json": JSONFormatter{},
}

// TextFormatter prints the result as human readable text.
type TextFormatter struct{}

func (f TextFormatter) PrintMatches(
	writer io.Writer, matchList []ruleMatch,
	showData bool, showAll bool, rowStr string,
) error {
	// print matches for table
	for _, match := range matchList {
		if showAll || match.Confidence != "low" {
			var description string

			count := len(match.MatchedData)
			if match.MatchType == "name" {
				description = fmt.Sprintf("possible %s (name match)", match.DisplayName)
			} else {
				str := pluralize(count, rowStr)
				if match.Confidence == "low" {
					str = str + ", low confidence"
				}
				description = fmt.Sprintf("found %s (%s)", match.DisplayName, str)
			}

			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Fprintf(writer, "%s %s\n", yellow(match.Identifier+":"), description)

			if showData {
				v := unique(match.MatchedData)
				if len(v) > 0 && showData {
					if len(v) > 50 {
						v = v[0:50]
					}

					for i, v2 := range v {
						v[i] = space.ReplaceAllString(v2, " ")
					}
					sort.Strings(v)
					fmt.Fprintln(writer, "    "+strings.Join(v, ", "))
				}
				fmt.Fprintln(writer, "")
			}
		}
	}

	return nil
}

// JSONFormatter prints the result as a JSON object.
type JSONFormatter struct{}

type jsonEntry struct {
	Name        string `json:"name"`
	Confidence  string `json:"confidence"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type jsonEntryWithMatches struct {
	jsonEntry

	Matches      []string `json:"matches"`
	MatchesCount int      `json:"matches_count"`
}

func (f JSONFormatter) PrintMatches(
	writer io.Writer, matchList []ruleMatch,
	showData bool, showAll bool, rowStr string,
) error {
	encoder := json.NewEncoder(writer)

	for _, match := range matchList {
		if showAll || match.Confidence != "low" {
			var description string

			count := len(match.MatchedData)
			if match.MatchType == "name" {
				description = fmt.Sprintf("possible %s (name match)", match.DisplayName)
			} else {
				str := pluralize(count, rowStr)
				if match.Confidence == "low" {
					str = str + ", low confidence"
				}
				description = fmt.Sprintf("found %s (%s)", match.DisplayName, str)
			}

			entry := jsonEntry{
				Name:        match.DisplayName,
				Confidence:  match.Confidence,
				Identifier:  match.Identifier,
				Description: description,
			}

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

					err := encoder.Encode(jsonEntryWithMatches{
						jsonEntry:    entry,
						Matches:      v,
						MatchesCount: len(v),
					})
					if err != nil {
						return err
					}
				}
			} else {
				err := encoder.Encode(entry)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
