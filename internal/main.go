package internal

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

func Main(urlStr string, showData bool, showAll bool, limit int, processes int, formatter Formatter) {
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
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Found %s to scan...\n", pluralize(len(files), "file")))

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
					err := formatter.PrintMatches(os.Stdout, fileMatchList, showData, showAll, "line")
					if err != nil {
						log.Fatal(err)
					}

					appendMutex.Lock()
					matchList = append(matchList, fileMatchList...)
					appendMutex.Unlock()

					<-guard
				}(f)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Found no files to scan")
			return
		}
	} else {
		var adapter Adapter = &SqlAdapter{}
		adapter.Init(urlStr)

		tables := adapter.FetchTables()

		if len(tables) > 0 {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Found %s to scan, sampling %d rows from each...\n", pluralize(len(tables), "table"), limit))

			wg.Add(len(tables))

			var queryMutex sync.Mutex

			for _, t := range tables {
				go func(t table, limit int) {
					defer wg.Done()

					queryMutex.Lock()
					columnNames, columnValues := adapter.FetchTableData(t, limit)
					queryMutex.Unlock()

					tableMatchList := checkTableData(t, columnNames, columnValues)
					err := formatter.PrintMatches(os.Stdout, tableMatchList, showData, showAll, "row")
					if err != nil {
						log.Fatal(err)
					}

					appendMutex.Lock()
					matchList = append(matchList, tableMatchList...)
					appendMutex.Unlock()
				}(t, limit)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Found no tables to scan")
			return
		}
	}

	wg.Wait()

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
}
