package internal

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type Adapter interface {
	Scan(scanOpts ScanOpts) ([]ruleMatch, error)
}

type ScanOpts struct {
	UrlStr      string
	ShowData    bool
	ShowAll     bool
	Limit       int
	Debug       bool
	Formatter   Formatter
	MatchConfig *MatchConfig
}

func Main(urlStr string, showData bool, showAll bool, limit int, processes int, only string, except string, minCount int, pattern string, debug bool, format string) error {
	runtime.GOMAXPROCS(processes)

	formatter, found := Formatters[format]
	if !found {
		arr := make([]string, 0, len(Formatters))
		for k := range Formatters {
			arr = append(arr, k)
		}
		sort.Strings(arr)
		return fmt.Errorf("Invalid format: %s\nValid formats are %s", format, strings.Join(arr, ", "))
	}

	matchConfig := NewMatchConfig()
	if pattern != "" {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		matchConfig.RegexRules = []regexRule{regexRule{Name: "pattern", DisplayName: "pattern", Confidence: "high", Regex: regex}}
		matchConfig.NameRules = matchConfig.NameRules[:0]
		matchConfig.MultiNameRules = matchConfig.MultiNameRules[:0]
		matchConfig.TokenRules = matchConfig.TokenRules[:0]
	} else {
		if except != "" {
			err := updateRules(&matchConfig, except, true)
			if err != nil {
				return err
			}
		}
		if only != "" {
			err := updateRules(&matchConfig, only, false)
			if err != nil {
				return err
			}
		}
	}
	matchConfig.MinCount = minCount

	var adapter Adapter
	if strings.HasPrefix(urlStr, "file://") {
		adapter = &LocalFileAdapter{}
	} else if strings.HasPrefix(urlStr, "s3://") {
		adapter = &S3Adapter{}
	} else if strings.HasPrefix(urlStr, "mongodb://") {
		adapter = &MongodbAdapter{}
	} else if strings.HasPrefix(urlStr, "redis://") {
		adapter = &RedisAdapter{}
	} else if strings.HasPrefix(urlStr, "elasticsearch+http://") || strings.HasPrefix(urlStr, "elasticsearch+https://") {
		adapter = &ElasticsearchAdapter{}
	} else if strings.HasPrefix(urlStr, "opensearch+http://") || strings.HasPrefix(urlStr, "opensearch+https://") {
		adapter = &ElasticsearchAdapter{}
	} else {
		adapter = &SqlAdapter{}
	}

	matchList, err := adapter.Scan(ScanOpts{urlStr, showData, showAll, limit, debug, formatter, &matchConfig})

	if err != nil {
		return err
	}

	if matchList == nil {
		return nil
	}

	if len(matchList) > 0 {
		if showData {
			fmt.Fprintln(os.Stderr, "Showing 50 unique values from each")
		} else {
			fmt.Fprintln(os.Stderr, "\nUse --show-data to view data")
		}

		if !showAll {
			showLowConfidenceMatchHelp(matchList)
		}
	} else {
		fmt.Fprintln(os.Stderr, "No sensitive data found")
	}

	return nil
}

func scanDataStore(adapter DataStoreAdapter, scanOpts ScanOpts) ([]ruleMatch, error) {
	err := adapter.Init(scanOpts.UrlStr)
	if err != nil {
		return nil, err
	}

	tables, err := adapter.FetchTables()
	if err != nil {
		return nil, err
	}

	if len(tables) > 0 {
		limit := scanOpts.Limit

		fmt.Fprintf(os.Stderr, "Found %s to scan, sampling %s from each...\n\n", pluralize(len(tables), adapter.TableName()), pluralize(limit, adapter.RowName()))

		matchList := []ruleMatch{}

		var g errgroup.Group
		var appendMutex sync.Mutex
		var queryMutex sync.Mutex

		for _, table := range tables {
			// important - do not remove
			// https://go.dev/doc/faq#closures_and_goroutines
			table := table

			g.Go(func() error {
				start := time.Now()

				// limit to one query at a time
				queryMutex.Lock()
				tableData, err := adapter.FetchTableData(table, limit)
				queryMutex.Unlock()

				if scanOpts.Debug {
					duration := time.Now().Sub(start)
					fmt.Fprintf(os.Stderr, "Scanned %s (%d ms)\n", table.displayName(), duration.Milliseconds())
				}

				if err != nil {
					return err
				}

				matchFinder := NewMatchFinder(scanOpts.MatchConfig)
				tableMatchList := matchFinder.CheckTableData(table, tableData)

				err = printMatchList(scanOpts.Formatter, tableMatchList, scanOpts.ShowData, scanOpts.ShowAll, adapter.RowName())
				if err != nil {
					return err
				}

				appendMutex.Lock()
				matchList = append(matchList, tableMatchList...)
				appendMutex.Unlock()

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}

		return matchList, nil
	} else {
		fmt.Fprintf(os.Stderr, "Found no %s to scan\n", pluralize(0, adapter.TableName())[2:])
		return nil, nil
	}
}

func scanFiles(adapter FileAdapter, scanOpts ScanOpts) ([]ruleMatch, error) {
	err := adapter.Init(scanOpts.UrlStr)
	if err != nil {
		return nil, err
	}

	files, err := adapter.FetchFiles()
	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		fmt.Fprintf(os.Stderr, "Found %s to scan...\n\n", pluralize(len(files), adapter.ObjectName()))

		matchList := []ruleMatch{}

		var g errgroup.Group
		var appendMutex sync.Mutex

		g.SetLimit(20)

		for _, file := range files {
			// important - do not remove
			// https://go.dev/doc/faq#closures_and_goroutines
			file := file

			g.Go(func() error {
				start := time.Now()

				matchFinder := NewMatchFinder(scanOpts.MatchConfig)
				err := adapter.FindFileMatches(file, &matchFinder)

				if scanOpts.Debug {
					duration := time.Now().Sub(start)
					fmt.Fprintf(os.Stderr, "Scanned %s (%d ms)\n", file, duration.Milliseconds())
				}

				if err != nil {
					return err
				}

				fileMatchList := matchFinder.CheckMatches(file, true)

				err = printMatchList(scanOpts.Formatter, fileMatchList, scanOpts.ShowData, scanOpts.ShowAll, "line")
				if err != nil {
					return err
				}

				appendMutex.Lock()
				matchList = append(matchList, fileMatchList...)
				appendMutex.Unlock()

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}

		return matchList, nil
	} else {
		fmt.Fprintf(os.Stderr, "Found no %s to scan\n", pluralize(0, adapter.ObjectName())[2:])
		return nil, nil
	}
}

func updateRules(matchConfig *MatchConfig, value string, except bool) error {
	names := make(map[string]bool)
	validNames := makeValidNames(matchConfig)

	for _, name := range strings.Split(value, ",") {
		if name == "last_name" {
			name = "surname"
		}
		if !validNames[name] {
			arr := make([]string, 0, len(validNames))
			for k := range validNames {
				arr = append(arr, k)
			}
			sort.Strings(arr)
			return fmt.Errorf("Invalid rule: %s\nValid rules are %s", name, strings.Join(arr, ", "))
		}
		names[name] = true
	}

	regexRules := []regexRule{}
	for _, rule := range matchConfig.RegexRules {
		var keep bool
		if except {
			keep = !names[rule.Name]
		} else {
			keep = names[rule.Name]
		}

		if keep {
			regexRules = append(regexRules, rule)
		}
	}
	matchConfig.RegexRules = regexRules

	nameRules := []nameRule{}
	for _, rule := range matchConfig.NameRules {
		var keep bool
		if except {
			keep = !names[rule.Name]
		} else {
			keep = names[rule.Name]
		}

		if keep {
			nameRules = append(nameRules, rule)
		}
	}
	matchConfig.NameRules = nameRules

	multiNameRules := []multiNameRule{}
	for _, rule := range matchConfig.MultiNameRules {
		var keep bool
		if except {
			keep = !names[rule.Name]
		} else {
			keep = names[rule.Name]
		}

		if keep {
			multiNameRules = append(multiNameRules, rule)
		}
	}
	matchConfig.MultiNameRules = multiNameRules

	tokenRules := []tokenRule{}
	for _, rule := range matchConfig.TokenRules {
		var keep bool
		if except {
			keep = !names[rule.Name]
		} else {
			keep = names[rule.Name]
		}

		if keep {
			tokenRules = append(tokenRules, rule)
		}
	}
	matchConfig.TokenRules = tokenRules

	return nil
}

func makeValidNames(matchConfig *MatchConfig) map[string]bool {
	validNames := make(map[string]bool)
	for _, rule := range matchConfig.RegexRules {
		validNames[rule.Name] = true
	}
	for _, rule := range matchConfig.NameRules {
		validNames[rule.Name] = true
	}
	for _, rule := range matchConfig.MultiNameRules {
		validNames[rule.Name] = true
	}
	for _, rule := range matchConfig.TokenRules {
		validNames[rule.Name] = true
	}
	return validNames
}
