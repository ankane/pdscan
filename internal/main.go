package internal

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

func Main(urlStr string, showData bool, showAll bool, limit int, processes int) error {
	runtime.GOMAXPROCS(processes)

	matchList := []ruleMatch{}

	var appendMutex sync.Mutex

	var wg sync.WaitGroup

	if strings.HasPrefix(urlStr, "file://") || strings.HasPrefix(urlStr, "s3://") {
		var adapter FileAdapter
		if strings.HasPrefix(urlStr, "file://") {
			adapter = &LocalFileAdapter{}
		} else {
			adapter = &S3Adapter{}
		}
		adapter.Init(urlStr)

		files := adapter.FetchFiles()

		if len(files) > 0 {
			fmt.Printf("Found %s to scan...\n\n", pluralize(len(files), "file"))

			wg.Add(len(files))

			// https://stackoverflow.com/a/25306241/1177228
			guard := make(chan struct{}, 20)

			for _, f := range files {
				guard <- struct{}{}

				go func(file string) {
					defer wg.Done()

					// fmt.Println("Scanning " + file + "...\n")
					matchedValues, count := adapter.FindFileMatches(file)
					fileMatchList := checkMatches(file, matchedValues, count, true)
					printMatchList(fileMatchList, showData, showAll, "line")

					appendMutex.Lock()
					matchList = append(matchList, fileMatchList...)
					appendMutex.Unlock()

					<-guard
				}(f)
			}
		} else {
			fmt.Println("Found no files to scan")
			return nil
		}
	} else {
		var adapter Adapter
		if strings.HasPrefix(urlStr, "mongodb://") {
			adapter = &MongodbAdapter{}
		} else if strings.HasPrefix(urlStr, "redis://") {
			adapter = &RedisAdapter{}
		} else if strings.HasPrefix(urlStr, "elasticsearch+http://") || strings.HasPrefix(urlStr, "elasticsearch+https://") || strings.HasPrefix(urlStr, "opensearch+http://") || strings.HasPrefix(urlStr, "opensearch+https://") {
			adapter = &ElasticsearchAdapter{}
		} else {
			adapter = &SqlAdapter{}
		}
		err := adapter.Init(urlStr)
		if err != nil {
			return err
		}

		tables := adapter.FetchTables()

		if len(tables) > 0 {
			fmt.Printf("Found %s to scan, sampling %s from each...\n\n", pluralize(len(tables), adapter.TableName()), pluralize(limit, adapter.RowName()))

			wg.Add(len(tables))

			var queryMutex sync.Mutex

			for _, t := range tables {
				go func(t table, limit int) {
					defer wg.Done()

					queryMutex.Lock()
					columnNames, columnValues := adapter.FetchTableData(t, limit)
					queryMutex.Unlock()

					tableMatchList := checkTableData(t, columnNames, columnValues)
					printMatchList(tableMatchList, showData, showAll, "row")

					appendMutex.Lock()
					matchList = append(matchList, tableMatchList...)
					appendMutex.Unlock()
				}(t, limit)
			}
		} else {
			fmt.Printf("Found no %s to scan\n", pluralize(0, adapter.TableName())[2:])
			return nil
		}
	}

	wg.Wait()

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
