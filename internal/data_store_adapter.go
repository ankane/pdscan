package internal

type DataStoreAdapter interface {
	TableName() string
	RowName() string
	Init(url string) error
	FetchTables() ([]table, error)
	FetchTableData(table table, limit int) (*tableData, error)
}
