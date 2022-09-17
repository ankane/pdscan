package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	elasticsearch "github.com/opensearch-project/opensearch-go"
	esapi "github.com/opensearch-project/opensearch-go/opensearchapi"
)

type ElasticsearchAdapter struct {
	DB      *elasticsearch.Client
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
	if strings.HasPrefix(urlStr, "elasticsearch+") {
		urlStr = strings.TrimPrefix(urlStr, "elasticsearch+")
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
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}

	a.DB = es

	return nil
}

func (a ElasticsearchAdapter) FetchTables() ([]table, error) {
	tables := []table{}

	es := a.DB

	var r []interface{}

	res, err := es.Cat.Indices(
		es.Cat.Indices.WithIndex([]string{a.indices}...),
		es.Cat.Indices.WithS("index"),
		es.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = checkResult(res)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing the response body: %s", err)
	}

	for _, index := range r {
		indexName := index.(map[string]interface{})["index"].(string)

		// skip system indices
		if indexName[0] != '.' {
			tables = append(tables, table{Schema: "", Name: indexName})
		}
	}

	return tables, nil
}

func (a ElasticsearchAdapter) FetchTableData(table table, limit int) (*tableData, error) {
	es := a.DB

	var r map[string]interface{}

	// TODO sample
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": limit,
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := es.Search(
		es.Search.WithIndex(table.Name),
		es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = checkResult(res)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing the response body: %s", err)
	}

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		// TODO check _id
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		keyMap, columnValues = scanSource(source, "", keyMap, columnValues)
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return &tableData{columnNames, columnValues}, nil
}

func scanSource(object map[string]interface{}, prefix string, keyMap map[string]int, columnValues [][]string) (map[string]int, [][]string) {
	for key, val := range object {
		key = prefix + key
		i, ok := keyMap[key]
		if !ok {
			i = len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})
		}

		switch typedVal := val.(type) {
		case map[string]interface{}:
			keyMap, columnValues = scanSource(typedVal, key+".", keyMap, columnValues)
		case []interface{}:
			values := []string{}
			for _, av := range typedVal {
				switch av2 := av.(type) {
				case map[string]interface{}:
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

func checkResult(res *esapi.Response) error {
	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return err
		} else {
			return fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}
	return nil
}
