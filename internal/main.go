package internal

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Adapter interface {
	Scan(string, bool, bool, int, *MatchConfig) ([]ruleMatch, error)
}

func Main(urlStr string, showData bool, showAll bool, limit int, processes int, only string, except string, minCount int, pattern string) error {
	runtime.GOMAXPROCS(processes)

	matchConfig := NewMatchConfig()
	if pattern != "" {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		matchConfig.RegexRules = []regexRule{regexRule{Name: "pattern", DisplayName: "pattern", Regex: regex}}
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

	matchList, err := adapter.Scan(urlStr, showData, showAll, limit, &matchConfig)

	if err != nil {
		return err
	}

	if matchList == nil {
		return nil
	}

	if len(matchList) > 0 {
		if showData {
			fmt.Println("Showing 50 unique values from each")
		} else {
			fmt.Println("\nUse --show-data to view data")
		}

		if !showAll {
			showLowConfidenceMatchHelp(matchList)
		}
	} else {
		fmt.Println("No sensitive data found")
	}

	return nil
}

func scanDataStore(adapter DataStoreAdapter, urlStr string, showData bool, showAll bool, limit int, matchConfig *MatchConfig) ([]ruleMatch, error) {
	err := adapter.Init(urlStr)
	if err != nil {
		return nil, err
	}

	tables, err := adapter.FetchTables()
	if err != nil {
		return nil, err
	}

	if len(tables) > 0 {
		fmt.Printf("Found %s to scan, sampling %s from each...\n\n", pluralize(len(tables), adapter.TableName()), pluralize(limit, adapter.RowName()))

		matchList := []ruleMatch{}

		var g errgroup.Group
		var appendMutex sync.Mutex
		var queryMutex sync.Mutex

		for _, table := range tables {
			// important - do not remove
			// https://go.dev/doc/faq#closures_and_goroutines
			table := table

			g.Go(func() error {
				// limit to one query at a time
				queryMutex.Lock()
				tableData, err := adapter.FetchTableData(table, limit)
				queryMutex.Unlock()
				if err != nil {
					return err
				}

				matchFinder := NewMatchFinder(matchConfig)
				tableMatchList := matchFinder.CheckTableData(table, tableData)
				printMatchList(tableMatchList, showData, showAll, adapter.RowName())

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
		fmt.Printf("Found no %s to scan\n", pluralize(0, adapter.TableName())[2:])
		return nil, nil
	}
}

func scanFiles(adapter FileAdapter, urlStr string, showData bool, showAll bool, matchConfig *MatchConfig) ([]ruleMatch, error) {
	err := adapter.Init(urlStr)
	if err != nil {
		return nil, err
	}

	files, err := adapter.FetchFiles()
	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		fmt.Printf("Found %s to scan...\n\n", pluralize(len(files), adapter.ObjectName()))

		matchList := []ruleMatch{}

		var g errgroup.Group
		var appendMutex sync.Mutex

		g.SetLimit(20)

		for _, file := range files {
			// important - do not remove
			// https://go.dev/doc/faq#closures_and_goroutines
			file := file

			g.Go(func() error {
				// fmt.Println("Scanning " + file + "...\n")
				matchFinder := NewMatchFinder(matchConfig)
				err := adapter.FindFileMatches(file, &matchFinder)
				if err != nil {
					return err
				}
				fileMatchList := matchFinder.CheckMatches(file, true)
				printMatchList(fileMatchList, showData, showAll, "line")

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
		fmt.Printf("Found no %s to scan\n", pluralize(0, adapter.ObjectName())[2:])
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
