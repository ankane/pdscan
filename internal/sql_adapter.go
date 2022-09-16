package internal

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/xo/dburl"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

type SqlAdapter struct {
	DB *sqlx.DB
}

func (a *SqlAdapter) TableName() string {
	return "table"
}

func (a *SqlAdapter) RowName() string {
	return "row"
}

func (a *SqlAdapter) Scan(urlStr string, showData bool, showAll bool, limit int, matchConfig *MatchConfig) ([]ruleMatch, error) {
	return scanDataStore(a, urlStr, showData, showAll, limit, matchConfig)
}

func (a *SqlAdapter) Init(url string) error {
	u, err := dburl.Parse(url)
	if err != nil {
		return err
	}

	db, err := sqlx.Connect(u.Driver, u.DSN)
	if err != nil {
		// TODO prompt for password if needed
		// var input string
		// fmt.Scanln(&input)
		return err
	}
	// defer db.Close()

	a.DB = db

	return nil
}

func (a SqlAdapter) FetchTables() ([]table, error) {
	tables := []table{}

	db := a.DB

	var query string

	switch db.DriverName() {
	case "sqlite3":
		query = `SELECT '' AS table_schema, name AS table_name FROM sqlite_master WHERE type = 'table' AND name != 'sqlite_sequence' ORDER BY name`
	case "mysql":
		query = `SELECT table_schema AS table_schema, table_name AS table_name FROM information_schema.tables WHERE table_schema = DATABASE() OR (DATABASE() IS NULL AND table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')) ORDER BY table_schema, table_name`
	case "sqlserver":
		query = `SELECT table_schema, table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' ORDER BY table_schema, table_name`
	default:
		query = `SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'pg_catalog') ORDER BY table_schema, table_name`
	}

	err := db.Select(&tables, query)
	if err != nil {
		return nil, err
	}

	return tables, nil
}

func (a SqlAdapter) FetchTableData(table table, limit int) (*tableData, error) {
	db := a.DB

	var sql string
	if db.DriverName() == "postgres" {
		quotedTable := quoteIdent(table.Schema) + "." + quoteIdent(table.Name)

		if tsmSystemRowsSupported(db) {
			sql = fmt.Sprintf("SELECT * FROM %s TABLESAMPLE SYSTEM_ROWS(%d)", quotedTable, limit)
		} else {
			// TODO randomize
			sql = fmt.Sprintf("SELECT * FROM %s LIMIT %d", quotedTable, limit)
		}
	} else if db.DriverName() == "sqlite3" {
		// TODO quote table name
		// TODO make more efficient if primary key exists
		// https://stackoverflow.com/questions/1253561/sqlite-order-by-rand
		sql = fmt.Sprintf("SELECT * FROM %s ORDER BY RANDOM() LIMIT %d", table.Name, limit)
	} else if db.DriverName() == "sqlserver" {
		// TODO quote table name
		sql = fmt.Sprintf("SELECT * FROM %s TABLESAMPLE (%d rows)", table.Name, limit)
	} else {
		// TODO quote table name
		// mysql
		sql = fmt.Sprintf("SELECT * FROM %s LIMIT %d", table.Schema+"."+table.Name, limit)
	}

	// run query on each table
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}

	// read everything as string and discard empty strings
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// map types
	columnNames := make([]string, len(cols))
	columnTypes := make([]string, len(cols))

	for i, col := range cols {
		columnNames[i] = col.Name()
		columnTypes[i] = col.DatabaseTypeName()
	}

	// check values
	rawResult := make([][]byte, len(cols))

	columnValues := make([][]string, len(cols))
	for i := range columnValues {
		columnValues[i] = []string{}
	}

	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
	for i := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			return nil, err
		}

		for i, raw := range rawResult {
			if raw == nil {
				// ignore
			} else {
				str := string(raw)
				if str != "" {
					columnValues[i] = append(columnValues[i], str)
				}
			}
		}
	}

	return &tableData{columnNames, columnValues}, nil
}

// helpers

func quoteIdent(column string) string {
	return pq.QuoteIdentifier(column)
}

func tsmSystemRowsSupported(db *sqlx.DB) bool {
	row := db.QueryRow("SELECT COUNT(*) FROM pg_extension WHERE extname = 'tsm_system_rows'")
	var count int
	err := row.Scan(&count)
	if err != nil {
		// redshift
		return false
	}
	return count > 0
}
