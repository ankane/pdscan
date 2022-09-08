package internal

type Adapter interface {
	TableName() string
	RowName() string
	Init(url string) error
	FetchTables() (tables []table)
	FetchTableData(table table, limit int) ([]string, [][]string)
}
