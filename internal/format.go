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
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Fprintf(writer, "%s %s\n", yellow(match.Identifier+":"), match.Description)

	values := match.Values
	if values != nil {
		if len(values) > 0 {
			fmt.Fprintln(writer, "    "+strings.Join(values, ", "))
		}
		fmt.Fprintln(writer, "")
	}
	return nil
}

// JSONFormatter prints the result as a JSON object.
type JSONFormatter struct{}

type jsonEntry struct {
	Name        string `json:"name"`
	MatchType   string `json:"match_type"`
	Confidence  string `json:"confidence"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type jsonEntryWithMatches struct {
	jsonEntry

	Matches      []string `json:"matches"`
	MatchesCount int      `json:"matches_count"`
}

func (f JSONFormatter) PrintMatch(writer io.Writer, match matchInfo) error {
	encoder := json.NewEncoder(writer)

	entry := jsonEntry{
		Name:        match.RuleName,
		MatchType:   match.MatchType,
		Confidence:  match.Confidence,
		Identifier:  match.Identifier,
		Description: match.Description,
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
