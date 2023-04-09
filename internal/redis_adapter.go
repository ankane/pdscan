package internal

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
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

func (a *RedisAdapter) Scan(scanOpts ScanOpts) ([]ruleMatch, error) {
	return scanDataStore(a, scanOpts)
}

func (a *RedisAdapter) Init(urlStr string) error {
	opt, err := redis.ParseURL(urlStr)
	if err != nil {
		return err
	}

	a.DB = redis.NewClient(opt)

	// connect
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = a.DB.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}

func (a RedisAdapter) FetchTables() ([]table, error) {
	return []table{{Schema: "", Name: ""}}, nil
}

func (a RedisAdapter) FetchTableData(table table, limit int) (*tableData, error) {
	rdb := a.DB

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keyMap := make(map[string]int)

	columnValues := make([][]string, 0)

	for j := 0; j < limit; j++ {
		key, err := rdb.RandomKey(ctx).Result()
		if err == redis.Nil {
			break
		} else if err != nil {
			return nil, err
		}

		_, ok := keyMap[key]
		if !ok {
			i := len(keyMap)
			keyMap[key] = i
			columnValues = append(columnValues, []string{})

			ty, err := rdb.Type(ctx, key).Result()
			if err != nil {
				return nil, err
			}

			if ty == "string" {
				val, err := rdb.Get(ctx, key).Result()
				if err != nil {
					return nil, err
				}
				columnValues[i] = append(columnValues[i], val)
			} else if ty == "list" {
				// no LSCAN
				// https://github.com/redis/redis/issues/6538
				// TODO fetch in batches
				val, err := rdb.LRange(ctx, key, 0, 1000).Result()
				if err != nil {
					return nil, err
				}
				columnValues[i] = append(columnValues[i], val...)
			} else if ty == "set" {
				iter := rdb.SScan(ctx, key, 0, "", 0).Iterator()
				for iter.Next(ctx) {
					v := iter.Val()
					columnValues[i] = append(columnValues[i], v)
				}
				if err := iter.Err(); err != nil {
					return nil, err
				}
			} else if ty == "hash" {
				iter := rdb.HScan(ctx, key, 0, "", 0).Iterator()
				for iter.Next(ctx) {
					v := iter.Val()
					columnValues[i] = append(columnValues[i], v)
				}
				if err := iter.Err(); err != nil {
					return nil, err
				}
			} else if ty == "zset" {
				iter := rdb.ZScan(ctx, key, 0, "", 0).Iterator()
				for iter.Next(ctx) {
					v := iter.Val()
					columnValues[i] = append(columnValues[i], v)
				}
				if err := iter.Err(); err != nil {
					return nil, err
				}
			}
		}
	}

	columnNames := make([]string, len(keyMap))
	for key, i := range keyMap {
		columnNames[i] = key
	}

	return &tableData{columnNames, columnValues}, nil
}
