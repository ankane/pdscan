package internal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

func (a *MongodbAdapter) Init(urlStr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(urlStr))
	if err != nil {
		abort(err)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		abort(err)
	}

	if len(u.Path) < 2 {
		abort(errors.New("No database specified"))
	}

	a.DB = client.Database(u.Path[1:])
}

func (a MongodbAdapter) FetchTables() (tables []table) {
	db := a.DB

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.ListCollectionNames(ctx, bson.D{{}})
	if err != nil {
		abort(err)
	}

	for _, name := range result {
		tables = append(tables, table{Schema: "", Name: name})
	}

	return tables
}

func (a MongodbAdapter) FetchTableData(table table, limit int) ([]string, [][]string) {
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
		abort(err)
	}
	defer cur.Close(ctx)

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil {
			abort(err)
		}

		keyMap, columnValues = scanObject(result, "", keyMap, columnValues)
	}
	if err := cur.Err(); err != nil {
		abort(err)
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return columnNames, columnValues
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
					for _, av := range arr {
						str, ok := av.(string)
						if ok {
							columnValues[i] = append(columnValues[i], str)
						}
					}
				}
			}
		}
	}
	return keyMap, columnValues
}
