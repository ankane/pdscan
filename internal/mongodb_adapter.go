package internal

import (
	"context"
	"errors"
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

		for _, elem := range result {
			i, ok := keyMap[elem.Key]
			if !ok {
				i = len(keyMap)
				keyMap[elem.Key] = i
				columnValues = append(columnValues, []string{})
			}

			// TODO scan nested values
			str, ok := elem.Value.(string)
			if ok {
				columnValues[i] = append(columnValues[i], str)
			}
		}
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
