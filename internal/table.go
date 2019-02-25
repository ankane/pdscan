package internal

type table struct {
	Schema string `db:"table_schema"`
	Name   string `db:"table_name"`
}

func (t table) displayName() string {
	str := t.Name
	if t.Schema != "" {
		str = t.Schema + "." + str
	}
	return str
}
