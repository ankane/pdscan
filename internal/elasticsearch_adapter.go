package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"strings"

	elasticsearch "github.com/opensearch-project/opensearch-go/v4"
	esapi "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type ElasticsearchAdapter struct {
	client  *esapi.Client
	indices string
}

func (a *ElasticsearchAdapter) TableName() string {
	return "index"
}

func (a *ElasticsearchAdapter) RowName() string {
	return "document"
}

func (a *ElasticsearchAdapter) Scan(scanOpts ScanOpts) ([]ruleMatch, error) {
	return scanDataStore(a, scanOpts)
}

func (a *ElasticsearchAdapter) Init(urlStr string) error {
	if after, ok := strings.CutPrefix(urlStr, "elasticsearch+"); ok {
		urlStr = after
	} else {
		urlStr = strings.TrimPrefix(urlStr, "opensearch+")
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	// TODO keep path before last slash
	if len(u.Path) < 2 {
		a.indices = "_all"
	} else {
		a.indices = u.Path[1:]
	}
	u.Path = ""

	cfg := elasticsearch.Config{
		Addresses: []string{
			u.String(),
		},
	}
	client, err := esapi.NewClient(esapi.Config{Client: cfg})
	if err != nil {
		return err
	}

	a.client = client

	return nil
}

func (a ElasticsearchAdapter) FetchTables() ([]table, error) {
	tables := []table{}

	ctx := context.TODO()
	res, err := a.client.Cat.Indices(
		ctx,
		&esapi.CatIndicesReq{
			Indices: []string{a.indices},
			Params:  esapi.CatIndicesParams{Sort: []string{"index"}},
		},
	)
	if err != nil {
		return nil, err
	}

	for _, index := range res.Indices {
		indexName := index.Index

		// skip system indices
		if indexName[0] != '.' {
			tables = append(tables, table{Schema: "", Name: indexName})
		}
	}

	return tables, nil
}

func (a ElasticsearchAdapter) FetchTableData(table table, limit int) (*tableData, error) {
	// TODO sample
	var buf bytes.Buffer
	query := map[string]any{
		"query": map[string]any{
			"match_all": map[string]any{},
		},
		"size": limit,
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	res, err := a.client.Search(
		ctx,
		&esapi.SearchReq{
			Indices: []string{table.Name},
			Body:    &buf,
		},
	)
	if err != nil {
		return nil, err
	}

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for _, hit := range res.Hits.Hits {
		// TODO check _id
		var source map[string]any
		if err := json.Unmarshal(hit.Source, &source); err != nil {
			return nil, err
		}
		keyMap, columnValues = scanSource(source, "", keyMap, columnValues)
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return &tableData{columnNames, columnValues}, nil
}

func scanSource(object map[string]any, prefix string, keyMap map[string]int, columnValues [][]string) (map[string]int, [][]string) {
	for key, val := range object {
		key = prefix + key
		i, ok := keyMap[key]
		if !ok {
			i = len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})
		}

		switch typedVal := val.(type) {
		case map[string]any:
			keyMap, columnValues = scanSource(typedVal, key+".", keyMap, columnValues)
		case []any:
			values := []string{}
			for _, av := range typedVal {
				switch av2 := av.(type) {
				case map[string]any:
					keyMap, columnValues = scanSource(av2, key+".", keyMap, columnValues)
				case string:
					values = append(values, av2)
				}
			}
			// add as single value for now for correct document count
			if len(values) > 0 {
				columnValues[i] = append(columnValues[i], strings.Join(values, ", "))
			}
		case string:
			columnValues[i] = append(columnValues[i], typedVal)
		}
	}
	return keyMap, columnValues
}
