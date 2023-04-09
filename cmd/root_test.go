package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	elasticsearch "github.com/opensearch-project/opensearch-go"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestFileCsv(t *testing.T) {
	checkFile(t, "email.csv", true)
}

func TestFileCsvLocation(t *testing.T) {
	// TODO check column names
	checkFile(t, "location.csv", false)
}

func TestFileGit(t *testing.T) {
	stdout, _ := fileOutput("../.git")
	assert.Contains(t, stdout, ".git/logs/HEAD:")
}

func TestFileNoExt(t *testing.T) {
	checkFile(t, "email", true)
}

func TestFileTxt(t *testing.T) {
	checkFile(t, "email.txt", true)
}

func TestFileEmpty(t *testing.T) {
	checkFile(t, "empty.txt", false)
}

func TestFileMissing(t *testing.T) {
	_, stderr := fileOutput("missing.txt")
	assert.Contains(t, stderr, "Found no files to scan")
}

func TestFileTarGz(t *testing.T) {
	checkFile(t, "email.tar.gz", true)
}

func TestFileXlsx(t *testing.T) {
	checkFile(t, "email.xlsx", true)
}

func TestFileZip(t *testing.T) {
	checkFile(t, "email.zip", true)
}

func TestFileMinCount(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("min-count.txt"), "--min-count", "2"}) })
	assert.Contains(t, stdout, "found emails (2 lines)")

	_, stderr := captureOutput(func() { runCmd([]string{fileUrl("min-count.txt"), "--min-count", "3"}) })
	assert.Contains(t, stderr, "No sensitive data found")
}

func TestFileLineCount(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("min-count.txt"), "--show-data"}) })
	assert.Contains(t, stdout, "found emails (2 lines)")
	assert.Contains(t, stdout, "test1@example.org, test2@example.org, test3@example.org")
}

func TestUrl(t *testing.T) {
	checkFile(t, "url.txt", false)
}

func TestElasticsearch(t *testing.T) {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		panic(err)
	}

	_, err = es.Indices.Delete([]string{"pdscan_test_users"})
	if err != nil {
		panic(err)
	}

	str := `
		{
			"email": "test@example.org",
			"phone": "555-555-5555",
			"street": "123 Main St",
			"zip_code": "12345",
			"ip": "127.0.0.1",
			"ip2": "127.0.0.1",
			"birthday": "1970-01-01",
			"latitude": 1.2,
			"longitude": 3.4,
			"access_token": "secret",
			"emails": ["first@example.org", "second@example.org"],
			"nested": {
				"email": "test@example.org",
				"zip_code": "12345"
			},
			"nested_type": [
				{
					"id": 1,
					"email": "nested1@example.org"
				},
				{
					"id": 2,
					"email": "nested2@example.org"
				}
			]
		}
	`

	// TODO create separate documents like MongoDB
	res, err := es.Index(
		"pdscan_test_users",
		strings.NewReader(str),
		es.Index.WithDocumentID("1"),
		es.Index.WithRefresh("true"),
	)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	stdout, _ := checkDocument(t, "elasticsearch+http://localhost:9200/pdscan_test_*")
	assert.Contains(t, stdout, "users.nested_type.email:")
}

func TestMongodb(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	collection := client.Database("pdscan_test").Collection("users")
	if err = collection.Drop(ctx); err != nil {
		panic(err)
	}

	docs := []interface{}{
		bson.D{{"email", "test@example.org"}},
		bson.D{{"phone", "555-555-5555"}},
		bson.D{{"street", "123 Main St"}, {"zip_code", "12345"}},
		bson.D{{"ip", "127.0.0.1"}, {"ip2", "127.0.0.1"}},
		bson.D{{"birthday", "1970-01-01"}},
		bson.D{{"latitude", 1.2}, {"longitude", 3.4}},
		bson.D{{"access_token", "secret"}},
		bson.D{{"emails", bson.A{"first@example.org", "second@example.org"}}},
		bson.D{{"nested", bson.D{{"email", "test@example.org"}, {"zip_code", "12345"}}}},
	}
	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		panic(err)
	}

	checkDocument(t, "mongodb://localhost:27017/pdscan_test")
}

func TestMysql(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	db := setupDb("mysql", fmt.Sprintf("%s@/pdscan_test", currentUser.Username))
	db.MustExec(`
		CREATE TABLE users (
			id serial PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
			last_name text,
			ip varchar(15),
			ip2 varchar(15),
			latitude float,
			longitude float,
			access_token text
		)
	`)
	db.MustExec("INSERT INTO users (email, phone, street, ip, ip2) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1')")

	db.MustExec("DROP TABLE IF EXISTS `ITEMS`")
	db.MustExec("CREATE TABLE `ITEMS` (`EMAIL` text, `ZipCode` text)")
	db.MustExec("INSERT INTO `ITEMS` (`EMAIL`) VALUES ('test@example.org')")

	checkSql(t, fmt.Sprintf("mysql://%s@localhost/pdscan_test", currentUser.Username))
}

func TestPostgres(t *testing.T) {
	db := setupDb("postgres", "dbname=pdscan_test sslmode=disable")
	db.MustExec("CREATE EXTENSION IF NOT EXISTS hstore")
	db.MustExec(`
		CREATE TABLE users (
			id serial PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
			last_name text,
			ip inet,
			ip2 cidr,
			latitude float,
			longitude float,
			access_token text,
			mac macaddr,
			emails text[],
			settings json,
			settings2 jsonb,
			settings3 hstore
		)
	`)
	db.MustExec(`INSERT INTO users (email, phone, street, ip, ip2, mac, emails, settings, settings2, settings3) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1', 'a1:b2:c3:d4:e5:f6', ARRAY['array1@example.org', 'array2@example.org'], '{"email": "json@example.org"}', '{"email": "jsonb@example.org"}', 'email=>hstore@example.org')`)

	db.MustExec(`DROP TABLE IF EXISTS "ITEMS"`)
	db.MustExec(`CREATE TABLE "ITEMS" ("EMAIL" text, "ZipCode" text)`)
	db.MustExec(`INSERT INTO "ITEMS" ("EMAIL") VALUES ('test@example.org')`)

	stdout, _ := checkSql(t, "postgres://localhost/pdscan_test?sslmode=disable")
	assert.Contains(t, stdout, "users.mac:")
	assert.Contains(t, stdout, "users.emails: found emails (1 row)")
	assert.Contains(t, stdout, "users.settings:")
	assert.Contains(t, stdout, "users.settings2:")
	assert.Contains(t, stdout, "users.settings3:")
}

func TestRedis(t *testing.T) {
	var ctx = context.Background()

	urlStr := "redis://localhost:6379/1"
	opt, err := redis.ParseURL(urlStr)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opt)

	iter := rdb.Scan(ctx, 0, "pdscan_test:*", 0).Iterator()
	for iter.Next(ctx) {
		rdb.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}

	err = rdb.Set(ctx, "pdscan_test:email", "test@example.org", 0).Err()
	if err != nil {
		panic(err)
	}

	err = rdb.LPush(ctx, "pdscan_test:list", []string{"list1@example.org", "list2@example.org"}).Err()
	if err != nil {
		panic(err)
	}

	err = rdb.SAdd(ctx, "pdscan_test:set", []string{"set1@example.org", "set2@example.org"}).Err()
	if err != nil {
		panic(err)
	}

	err = rdb.HSet(ctx, "pdscan_test:hash", "email", "hash@example.org").Err()
	if err != nil {
		panic(err)
	}

	err = rdb.ZAdd(ctx, "pdscan_test:zset", redis.Z{Member: "zset1@example.org", Score: 1}).Err()
	if err != nil {
		panic(err)
	}

	err = rdb.ZAdd(ctx, "pdscan_test:zset", redis.Z{Member: "zset2@example.org", Score: 2}).Err()
	if err != nil {
		panic(err)
	}

	stdout, stderr := captureOutput(func() { runCmd([]string{urlStr, "--show-data"}) })
	assert.Contains(t, stderr, "sampling 10000 keys")
	assert.Contains(t, stdout, "pdscan_test:email:")

	// lists
	assert.Contains(t, stdout, "pdscan_test:list:")
	assert.Contains(t, stdout, "list1@example.org")
	assert.Contains(t, stdout, "list2@example.org")

	// sets
	assert.Contains(t, stdout, "pdscan_test:set:")
	assert.Contains(t, stdout, "set1@example.org")
	assert.Contains(t, stdout, "set2@example.org")

	// hashes
	assert.Contains(t, stdout, "pdscan_test:hash:")
	assert.Contains(t, stdout, "hash@example.org")

	// sorted sets
	assert.Contains(t, stdout, "pdscan_test:zset:")
	assert.Contains(t, stdout, "zset1@example.org")
	assert.Contains(t, stdout, "zset2@example.org")
}

func TestSqlite(t *testing.T) {
	dir, err := os.MkdirTemp("", "pdscan")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "test.sqlite3")
	db := setupDb("sqlite3", path)
	db.MustExec(`
		CREATE TABLE users (
			id serial PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
			last_name text,
			ip text,
			ip2 text,
			latitude float,
			longitude float,
			access_token text
		)
	`)
	db.MustExec("INSERT INTO users (email, phone, street, ip, ip2) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1')")

	db.MustExec(`DROP TABLE IF EXISTS "ITEMS"`)
	db.MustExec(`CREATE TABLE "ITEMS" ("EMAIL" text, "ZipCode" text)`)
	db.MustExec(`INSERT INTO "ITEMS" ("EMAIL") VALUES ('test@example.org')`)

	checkSql(t, fmt.Sprintf("sqlite://%s", path))
}

func TestSqlserver(t *testing.T) {
	url := os.Getenv("SQLSERVER_URL")
	if url == "" {
		t.Skip("Requires SQLSERVER_URL")
	}

	db := setupDb("sqlserver", url)
	db.MustExec(`
		CREATE TABLE users (
			id int IDENTITY(1,1) PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
			last_name text,
			ip text,
			ip2 text,
			latitude float,
			longitude float,
			access_token text
		)
	`)
	db.MustExec("INSERT INTO users (email, phone, street, ip, ip2) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1')")

	db.MustExec(`DROP TABLE IF EXISTS "ITEMS"`)
	db.MustExec(`CREATE TABLE "ITEMS" ("EMAIL" text, "ZipCode" text)`)
	db.MustExec(`INSERT INTO "ITEMS" ("EMAIL") VALUES ('test@example.org')`)

	checkSql(t, url)
}

func TestBadScheme(t *testing.T) {
	err := runCmd([]string{"hello://"})
	assert.Contains(t, err, "unknown database scheme")
}

func TestPattern(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("min-count.txt"), "--pattern", `\stest[12]`, "--show-data"}) })
	assert.Contains(t, stdout, "found pattern (1 line)")
	assert.NotContains(t, stdout, "found email")
	assert.NotContains(t, stdout, "test1")
	assert.Contains(t, stdout, "test2")
	assert.NotContains(t, stdout, "test3")
}

func TestBadPattern(t *testing.T) {
	err := runCmd([]string{fileUrl("min-count.txt"), "--pattern", `\e`})
	assert.Contains(t, err.Error(), "error parsing regexp: invalid escape sequence: `\\e`")
}

func TestFormatNdjson(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("email.txt"), "--format", "ndjson"}) })
	assert.Contains(t, stdout, `"name":"email"`)
	assert.Contains(t, stdout, `"confidence":"high"`)
}

func TestFormatNdjsonShowData(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("email.txt"), "--format", "ndjson", "--show-data"}) })
	assert.Contains(t, stdout, `"matches":["test@example.org"]`)
}

func TestFormatNdjsonShowAll(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("email.txt"), "--format", "ndjson", "--show-all"}) })
	// TODO test low confidence
	assert.Contains(t, stdout, `"confidence":"high"`)
}

func TestBadFormat(t *testing.T) {
	err := runCmd([]string{fileUrl("email.txt"), "--format", "bad"})
	assert.Contains(t, err.Error(), "Invalid format: bad")
	assert.Contains(t, err.Error(), "Valid formats are ndjson, text")
}

func TestShowData(t *testing.T) {
	stdout, _ := captureOutput(func() { runCmd([]string{fileUrl("email.txt"), "--show-data"}) })
	assert.Contains(t, stdout, "test@example.org")
}

// TODO uncomment in 0.2.0
// func TestExtraArgs(t *testing.T) {
// 	err := runCmd([]string{fileUrl("email.txt"), "other"})
// 	assert.Contains(t, err.Error(), "Too many arguments")
// }

// TODO fix
// func TestSampleSize(t *testing.T) {
// 	output := captureOutput(func() { internal.Main("sqlite:../testdata/test.sqlite3", false, false, 250, 1) })
// 	assert.Contains(t, output, "sampling 250 rows from each")
// }

// helpers

func captureOutput(f func()) (string, string) {
	color.NoColor = true
	stdout := os.Stdout
	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	os.Stderr = w2
	f()
	w.Close()
	w2.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	out2, err := io.ReadAll(r2)
	if err != nil {
		panic(err)
	}
	os.Stdout = stdout
	os.Stderr = stderr
	color.NoColor = false
	return string(out), string(out2)
}

func runCmd(args []string) error {
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	return cmd.Execute()
}

func fileUrl(filename string) string {
	return fmt.Sprintf("file://../testdata/%s", filename)
}

func fileOutput(filename string) (string, string) {
	return captureOutput(func() { runCmd([]string{fileUrl(filename)}) })
}

func checkFile(t *testing.T, filename string, found bool) {
	stdout, stderr := fileOutput(filename)
	assert.Contains(t, stderr, "Found 1 file to scan...")
	if found {
		assert.Contains(t, stdout, fmt.Sprintf("%s:", filename))
	} else {
		assert.Contains(t, stderr, "No sensitive data found")
	}
}

func setupDb(driver string, dsn string) *sqlx.DB {
	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		panic(err)
	}
	db.MustExec("DROP TABLE IF EXISTS users")
	return db
}

func checkSql(t *testing.T, urlStr string) (string, string) {
	stdout, stderr := captureOutput(func() { runCmd([]string{urlStr}) })
	assert.Contains(t, stderr, "sampling 10000 rows")
	assert.NotContains(t, stdout, "users.id:")
	assert.Contains(t, stdout, "users.email:")
	assert.Contains(t, stdout, "users.phone:")
	assert.Contains(t, stdout, "users.street:")
	assert.Contains(t, stdout, "users.zip_code:")
	assert.Contains(t, stdout, "users.birthday:")
	assert.Contains(t, stdout, "users.last_name:")
	assert.Contains(t, stdout, "users.ip:")
	assert.Contains(t, stdout, "users.ip2:")
	assert.Contains(t, stdout, "users.latitude+longitude:")
	assert.Contains(t, stdout, "users.access_token:")
	assert.Contains(t, stdout, "ITEMS.EMAIL:")
	assert.Contains(t, stdout, "ITEMS.ZipCode:")

	checkOnly(t, urlStr)
	checkExcept(t, urlStr)

	return stdout, stderr
}

func checkDocument(t *testing.T, urlStr string) (string, string) {
	stdout, stderr := captureOutput(func() { runCmd([]string{urlStr, "--show-data"}) })
	assert.Contains(t, stderr, "sampling 10000 documents")
	assert.NotContains(t, stdout, "users._id:")
	assert.Contains(t, stdout, "users.email:")
	assert.Contains(t, stdout, "users.phone:")
	assert.Contains(t, stdout, "users.street:")
	assert.Contains(t, stdout, "users.zip_code:")
	assert.Contains(t, stdout, "users.birthday:")
	assert.Contains(t, stdout, "users.ip:")
	assert.Contains(t, stdout, "users.ip2:")
	assert.Contains(t, stdout, "users.latitude+longitude:")
	assert.Contains(t, stdout, "users.access_token:")

	// arrays
	assert.Contains(t, stdout, "users.emails: found emails (1 document)")
	assert.Contains(t, stdout, "first@example.org")
	assert.Contains(t, stdout, "second@example.org")

	// nested
	assert.Contains(t, stdout, "users.nested.email:")
	assert.Contains(t, stdout, "users.nested.zip_code:")
	return stdout, stderr
}

func checkOnly(t *testing.T, urlStr string) {
	stdout, _ := captureOutput(func() { runCmd([]string{urlStr, "--only", "email,postal_code,location"}) })
	assert.Contains(t, stdout, "users.email:")
	assert.NotContains(t, stdout, "users.phone:")
	assert.NotContains(t, stdout, "users.street:")
	assert.Contains(t, stdout, "users.zip_code:")
	assert.NotContains(t, stdout, "users.birthday:")
	assert.NotContains(t, stdout, "users.ip:")
	assert.NotContains(t, stdout, "users.ip2:")
	assert.Contains(t, stdout, "users.latitude+longitude:")
	assert.NotContains(t, stdout, "users.access_token:")

	err := runCmd([]string{urlStr, "--only", "email,phone2"})
	assert.Contains(t, err.Error(), "Invalid rule: phone2")
	assert.Contains(t, err.Error(), "Valid rules are credit_card, date_of_birth, email")
}

func checkExcept(t *testing.T, urlStr string) {
	stdout, _ := captureOutput(func() { runCmd([]string{urlStr, "--except", "email,postal_code,location"}) })
	assert.NotContains(t, stdout, "users.email:")
	assert.Contains(t, stdout, "users.phone:")
	assert.Contains(t, stdout, "users.street:")
	assert.NotContains(t, stdout, "users.zip_code:")
	assert.Contains(t, stdout, "users.birthday:")
	assert.Contains(t, stdout, "users.ip:")
	assert.Contains(t, stdout, "users.ip2:")
	assert.NotContains(t, stdout, "users.latitude+longitude:")
	assert.Contains(t, stdout, "users.access_token:")

	err := runCmd([]string{urlStr, "--except", "email,phone2"})
	assert.Contains(t, err.Error(), "Invalid rule: phone2")
	assert.Contains(t, err.Error(), "Valid rules are credit_card, date_of_birth, email")
}
