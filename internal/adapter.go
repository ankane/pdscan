package internal

type Adapter interface {
	Init(url string)
	FetchTables() (tables []table)
	FetchTableData(table table, limit int) ([]string, [][]string)
}
