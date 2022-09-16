package internal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbAdapter struct {
	DB *mongo.Database
}

func (a *MongodbAdapter) TableName() string {
	return "collection"
}

func (a *MongodbAdapter) RowName() string {
	return "document"
}

func (a *MongodbAdapter) Scan(urlStr string, showData bool, showAll bool, limit int, matchConfig *MatchConfig) ([]ruleMatch, error) {
	return scanDataStore(a, urlStr, showData, showAll, limit, matchConfig)
}

func (a *MongodbAdapter) Init(urlStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(urlStr))
	if err != nil {
		return err
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if len(u.Path) < 2 {
		return errors.New("no database specified")
	}

	a.DB = client.Database(u.Path[1:])

	return nil
}

func (a MongodbAdapter) FetchTables() ([]table, error) {
	tables := []table{}

	db := a.DB

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.ListCollectionNames(ctx, bson.D{{}})
	if err != nil {
		return nil, err
	}

	for _, name := range result {
		tables = append(tables, table{Schema: "", Name: name})
	}

	return tables, nil
}

func (a MongodbAdapter) FetchTableData(table table, limit int) (*tableData, error) {
	collection := a.DB.Collection(table.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// https://www.mongodb.com/docs/manual/reference/operator/aggregation/sample/
	pipeline := mongo.Pipeline{
		{
			{"$sample", bson.D{
				{"size", limit},
			}},
		},
	}
	cur, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil {
			return nil, err
		}

		keyMap, columnValues = scanObject(result, "", keyMap, columnValues)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return &tableData{columnNames, columnValues}, nil
}

func scanObject(object bson.D, prefix string, keyMap map[string]int, columnValues [][]string) (map[string]int, [][]string) {
	for _, elem := range object {
		key := fmt.Sprintf("%s%s", prefix, elem.Key)
		i, ok := keyMap[key]
		if !ok {
			i = len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})
		}

		// TODO improve code
		str, ok := elem.Value.(string)
		if ok {
			columnValues[i] = append(columnValues[i], str)
		} else {
			value, ok := elem.Value.(bson.D)
			if ok {
				keyMap, columnValues = scanObject(value, fmt.Sprintf("%s.", key), keyMap, columnValues)
			} else {
				arr, ok := elem.Value.(bson.A)
				if ok {
					values := []string{}
					for _, av := range arr {
						str, ok := av.(string)
						if ok {
							values = append(values, str)
						}
					}
					// add as single value for now for correct document count
					if len(values) > 0 {
						columnValues[i] = append(columnValues[i], strings.Join(values, ", "))
					}
				}
			}
		}
	}
	return keyMap, columnValues
}
