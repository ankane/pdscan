package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Format defines the interface used to deliver results to the end user.
type Formatter interface {
	// PrintMatch formats and prints the match to `writer`.
	PrintMatch(writer io.Writer, match matchInfo) error
}

// Formatters holds available formatters
var Formatters = map[string]Formatter{
	"text":   TextFormatter{},
	"ndjson": JSONFormatter{},
}

// TextFormatter prints the result as human readable text.
type TextFormatter struct{}

func (f TextFormatter) PrintMatch(writer io.Writer, match matchInfo) error {
	var description string
	if match.MatchType == "name" {
		description = fmt.Sprintf("possible %s (name match)", match.DisplayName)
	} else {
		str := pluralize(match.LineCount, match.RowStr)
		if match.Confidence == "low" {
			str = str + ", low confidence"
		}
		if match.RowStr == "key" {
			description = fmt.Sprintf("found %s", match.DisplayName)
		} else {
			description = fmt.Sprintf("found %s (%s)", match.DisplayName, str)
		}
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	_, err := fmt.Fprintf(writer, "%s %s\n", yellow(match.Identifier+":"), description)
	if err != nil {
		return err
	}

	values := match.Values
	if values != nil {
		// squish whitespace
		// TODO show whitespace
		for i, value := range values {
			values[i] = space.ReplaceAllString(value, " ")
		}

		if len(values) > 0 {
			_, err = fmt.Fprintln(writer, "    "+strings.Join(values, ", "))
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(writer, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// JSONFormatter prints the result as a JSON object.
type JSONFormatter struct{}

type jsonEntry struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	MatchType  string `json:"match_type"`
	Confidence string `json:"confidence"`
}

type jsonEntryWithMatches struct {
	jsonEntry

	Matches      []string `json:"matches"`
	MatchesCount int      `json:"matches_count"`
}

func (f JSONFormatter) PrintMatch(writer io.Writer, match matchInfo) error {
	encoder := json.NewEncoder(writer)

	entry := jsonEntry{
		Identifier: match.Identifier,
		Name:       match.RuleName,
		MatchType:  match.MatchType,
		Confidence: match.Confidence,
	}

	values := match.Values
	if values != nil {
		return encoder.Encode(jsonEntryWithMatches{
			jsonEntry:    entry,
			Matches:      values,
			MatchesCount: len(values),
		})
	} else {
		return encoder.Encode(entry)
	}
}
