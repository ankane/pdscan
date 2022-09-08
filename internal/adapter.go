package internal

type Adapter interface {
	TableName() string
	RowName() string
	Init(url string) error
	FetchTables() ([]table, error)
	FetchTableData(table table, limit int) ([]string, [][]string, error)
}
