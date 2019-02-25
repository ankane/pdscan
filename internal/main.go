package internal

import (
	"fmt"
)

func Main(url string, showData bool, showAll bool, limit int) {
	// TODO use URL scheme to determine non-SQL adapters
	var adapter Adapter = &SqlAdapter{}
	adapter.Init(url)

	tables := adapter.FetchTables()

	if len(tables) > 0 {
		fmt.Println(fmt.Sprintf("Found %s to scan, sampling %d rows from each...\n", pluralize(len(tables), "table"), limit))

		matchList := []ruleMatch{}
		for _, table := range tables {
			columnNames, columnValues := adapter.FetchTableData(table, limit)
			tableMatchList := checkTableData(table, columnNames, columnValues)
			matchList = append(matchList, tableMatchList...)
			printMatchList(tableMatchList, showData, showAll)
		}

		if len(matchList) > 0 {
			if showData {
				fmt.Println("Showing 50 unique values from each column")
			} else {
				fmt.Println("\nUse --show-data to view data")
			}

			if !showAll {
				showLowConfidenceMatchHelp(matchList)
			}
		} else {
			fmt.Println("No sensitive data found")
		}
	} else {
		fmt.Println("Found no tables to scan")
	}
}
