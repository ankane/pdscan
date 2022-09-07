package internal

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisAdapter struct {
	DB *redis.Client
}

func (a *RedisAdapter) TableName() string {
	return "database"
}

func (a *RedisAdapter) RowName() string {
	return "key"
}

func (a *RedisAdapter) Init(urlStr string) {
	opt, err := redis.ParseURL(urlStr)
	if err != nil {
		panic(err)
	}

	a.DB = redis.NewClient(opt)
}

func (a RedisAdapter) FetchTables() []table {
	return []table{{Schema: "", Name: ""}}
}

func (a RedisAdapter) FetchTableData(table table, limit int) ([]string, [][]string) {
	rdb := a.DB

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for i := 0; i < limit; i++ {
		key, err := rdb.RandomKey(ctx).Result()
		if err != nil {
			panic(err)
		}

		i, ok := keyMap[key]
		if !ok {
			i = len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})

			ty, err := rdb.Type(ctx, key).Result()
			if err != nil {
				panic(err)
			}

			// TODO support more types
			if ty == "string" {
				val, err := rdb.Get(ctx, key).Result()
				if err != nil {
					panic(err)
				}
				columnValues[i] = append(columnValues[i], val)
			}
		}
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return columnNames, columnValues
}
