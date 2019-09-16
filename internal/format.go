package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Format defines the interface used to deliver results to the end user.
type Formatter interface {
	// AddMatches is called when a formatter should handle new matches.
	//
	// This function should be safe for concurent use.
	AddMatches(
		matches []ruleMatch, showData bool, showAll bool, rowKind string,
	) error

	// Flush is called when the formatter should finish outputing any data it
	// may have buffered.
	Flush() error
}

// FormatterFactory
type FormatterFactory func(io.Writer) Formatter

// Formatters holds available formatters
var Formatters = map[string]FormatterFactory{
	"text": NewTextFormatter,
	"json": NewJSONFormatter,
}

// TextFormatter prints the result as human readable text.
type TextFormatter struct {
	io.Writer
}

func NewTextFormatter(out io.Writer) Formatter {
	return TextFormatter{
		Writer: out,
	}
}

func (f TextFormatter) AddMatches(
	matchList []ruleMatch, showData bool, showAll bool, rowStr string,
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
			fmt.Fprintf(f.Writer, "%s %s\n", yellow(match.Identifier+":"), description)

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
					fmt.Fprintln(f.Writer, "    "+strings.Join(v, ", "))
				}
				fmt.Fprintln(f.Writer, "")
			}
		}
	}

	return nil
}

func (f TextFormatter) Flush() error { return nil }

// JSONFormatter prints the result as a JSON object.
type JSONFormatter struct {
	sync.Mutex

	entries []interface{}
	encoder *json.Encoder
}

func NewJSONFormatter(out io.Writer) Formatter {
	return &JSONFormatter{
		entries: make([]interface{}, 0),
		encoder: json.NewEncoder(out),
	}
}

type JSONEntry struct {
	Name        string `json:"name"`
	Confidence  string `json:"confidence"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type jsonEntryWithMatches struct {
	JSONEntry

	Matches      []string `json:"matches"`
	MatchesCount int      `json:"matches_count"`
}

func (f *JSONFormatter) AddMatches(
	matchList []ruleMatch, showData bool, showAll bool, rowStr string,
) error {
	f.Lock()
	defer f.Unlock()

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

			entry := JSONEntry{
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

					f.entries = append(f.entries, jsonEntryWithMatches{
						JSONEntry:    entry,
						Matches:      v,
						MatchesCount: len(v),
					})

				}
			} else {
				f.entries = append(f.entries, entry)
			}
		}
	}

	return nil
}

func (f *JSONFormatter) Flush() error {
	return f.encoder.Encode(&f.entries)
}
