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

func (a *RedisAdapter) Init(urlStr string) error {
	opt, err := redis.ParseURL(urlStr)
	if err != nil {
		return err
	}

	a.DB = redis.NewClient(opt)

	return nil
}

func (a RedisAdapter) FetchTables() ([]table, error) {
	return []table{{Schema: "", Name: ""}}, nil
}

func (a RedisAdapter) FetchTableData(table table, limit int) ([]string, [][]string, error) {
	rdb := a.DB

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for i := 0; i < limit; i++ {
		key, err := rdb.RandomKey(ctx).Result()
		if err != nil {
			return nil, nil, err
		}

		i, ok := keyMap[key]
		if !ok {
			i = len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})

			ty, err := rdb.Type(ctx, key).Result()
			if err != nil {
				return nil, nil, err
			}

			if ty == "string" {
				val, err := rdb.Get(ctx, key).Result()
				if err != nil {
					return nil, nil, err
				}
				columnValues[i] = append(columnValues[i], val)
			} else if ty == "list" {
				// TODO fetch in batches
				val, err := rdb.LRange(ctx, key, 0, -1).Result()
				if err != nil {
					return nil, nil, err
				}
				for _, v := range val {
					columnValues[i] = append(columnValues[i], v)
				}
			} else if ty == "set" {
				val, err := rdb.SMembers(ctx, key).Result()
				if err != nil {
					return nil, nil, err
				}
				for _, v := range val {
					columnValues[i] = append(columnValues[i], v)
				}
			} else if ty == "hash" {
				val, err := rdb.HGetAll(ctx, key).Result()
				if err != nil {
					return nil, nil, err
				}
				for _, v := range val {
					columnValues[i] = append(columnValues[i], v)
				}
			} else if ty == "zset" {
				// TODO fetch in batches
				val, err := rdb.ZRange(ctx, key, 0, -1).Result()
				if err != nil {
					return nil, nil, err
				}
				for _, v := range val {
					columnValues[i] = append(columnValues[i], v)
				}
			}
		}
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return columnNames, columnValues, nil
}
