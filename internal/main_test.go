package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	elasticsearch "github.com/opensearch-project/opensearch-go"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestLastName(t *testing.T) {
	assertMatchName(t, "last_name", "last_name")
	assertMatchName(t, "last_name", "lname")
	assertMatchName(t, "last_name", "surname")
	assertMatchValues(t, "last_name", []string{"Smith"})
}

func TestEmail(t *testing.T) {
	assertMatchValues(t, "email", []string{"test@example.org"})
	refuteMatchValues(t, []string{"http://user:pass@example.org/hi"})
}

func TestIP(t *testing.T) {
	assertMatchValues(t, "ip", []string{"127.0.0.1"})
}

func TestAddress(t *testing.T) {
	assertMatchValues(t, "street", []string{"123 Main St"})
	assertMatchValues(t, "street", []string{"123 Main Street"})
	assertMatchValues(t, "street", []string{"123 Main Ave"})
	assertMatchValues(t, "street", []string{"123 Main Avenue"})
}

func TestPostalCode(t *testing.T) {
	assertMatchName(t, "postal_code", "zip")
	assertMatchName(t, "postal_code", "zipCode")
	assertMatchName(t, "postal_code", "postal_code")
}

func TestPhone(t *testing.T) {
	assertMatchValues(t, "phone", []string{"555-555-5555"})
	assertMatchName(t, "phone", "phone")
	assertMatchName(t, "phone", "phoneNumber")
	refuteMatchValues(t, []string{"5555555555"})

	// use 7 digit min
	// https://stackoverflow.com/questions/14894899/what-is-the-minimum-length-of-a-valid-international-phone-number
	refuteMatchValues(t, []string{"+123456"})
	assertMatchValues(t, "phone", []string{"+1234567"})
	assertMatchValues(t, "phone", []string{"+15555555555"})
	assertMatchValues(t, "phone", []string{"+123456789012345"})
	refuteMatchValues(t, []string{"+1234567890123456"})
}

func TestCreditCard(t *testing.T) {
	assertMatchValues(t, "credit_card", []string{"4242-4242-4242-4242"})
	assertMatchValues(t, "credit_card", []string{"4242 4242 4242 4242"})
	assertMatchValues(t, "credit_card", []string{"4242424242424242"})
	refuteMatchValues(t, []string{"0242424242424242"})
	refuteMatchValues(t, []string{"55555555-5555-5555-5555-555555555555"})
}

func TestSSN(t *testing.T) {
	assertMatchValues(t, "ssn", []string{"123-45-6789"})
	assertMatchValues(t, "ssn", []string{"123 45 6789"})
	refuteMatchValues(t, []string{"123456789"})
}

func TestDateOfBirth(t *testing.T) {
	assertMatchName(t, "date_of_birth", "dob")
	assertMatchName(t, "date_of_birth", "DateOfBirth")
	assertMatchName(t, "date_of_birth", "birthday")
}

func TestLocationData(t *testing.T) {
	assertMatchNames(t, "location", []string{"latitude", "longitude"})
	assertMatchNames(t, "location", []string{"lat", "lon"})
	assertMatchNames(t, "location", []string{"lat", "lng"})
}

func TestOAuthToken(t *testing.T) {
	assertMatchName(t, "oauth_token", "access_token")
	assertMatchName(t, "oauth_token", "refreshToken")
	assertMatchValues(t, "oauth_token", []string{"ya29.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
}

func TestMac(t *testing.T) {
	assertMatchValues(t, "mac", []string{"ff:ff:ff:ff:ff:ff"})
	assertMatchValues(t, "mac", []string{"a1:b2:c3:d4:e5:f6"})
	assertMatchValues(t, "mac", []string{"A1:B2:C3:D4:E5:F6"})
}

func TestFileCsv(t *testing.T) {
	checkFile(t, "email.csv", true)
}

func TestFileCsvLocation(t *testing.T) {
	// TODO check column names
	checkFile(t, "location.csv", false)
}

func TestFileGit(t *testing.T) {
	output := fileOutput("../.git")
	assert.Contains(t, output, ".git/logs/HEAD:")
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
	output := fileOutput("missing.txt")
	assert.Contains(t, output, "Found no files to scan")
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

func TestElasticsearch(t *testing.T) {
	es, _ := elasticsearch.NewDefaultClient()

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
			}

		}
	`

	// TODO drop index first
	// TODO create separate documents like MongoDB
	// TODO test/support nested type
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

	checkDocument(t, "elasticsearch+http://localhost:9200/pdscan_test_*")
}

func TestMongodb(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	collection := client.Database("pdscan_test").Collection("users")
	if err = collection.Drop(ctx); err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	checkDocument(t, "mongodb://localhost:27017/pdscan_test")
}

func TestMysql(t *testing.T) {
	currentUser, _ := user.Current()
	db := setupDb("mysql", fmt.Sprintf("%s@/pdscan_test", currentUser.Username))
	db.MustExec(`
		CREATE TABLE users (
			id serial PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
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
	db.MustExec(`
		CREATE TABLE users (
			id serial PRIMARY KEY,
			email varchar(255),
			phone char(20),
			street text,
			zip_code text,
			birthday date,
			ip inet,
			ip2 cidr,
			latitude float,
			longitude float,
			access_token text,
			mac macaddr
		)
	`)
	db.MustExec("INSERT INTO users (email, phone, street, ip, ip2, mac) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1', 'a1:b2:c3:d4:e5:f6')")

	db.MustExec(`DROP TABLE IF EXISTS "ITEMS"`)
	db.MustExec(`CREATE TABLE "ITEMS" ("EMAIL" text, "ZipCode" text)`)
	db.MustExec(`INSERT INTO "ITEMS" ("EMAIL") VALUES ('test@example.org')`)

	output := checkSql(t, "postgres://localhost/pdscan_test?sslmode=disable")
	assert.Contains(t, output, "users.mac:")
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

	output := captureOutput(func() { Main(urlStr, true, false, 10000, 1) })
	assert.Contains(t, output, "sampling 10000 keys")
	assert.Contains(t, output, "pdscan_test:email:")

	// lists
	assert.Contains(t, output, "pdscan_test:list:")
	assert.Contains(t, output, "list1@example.org")
	assert.Contains(t, output, "list2@example.org")

	// sets
	assert.Contains(t, output, "pdscan_test:set:")
	assert.Contains(t, output, "set1@example.org")
	assert.Contains(t, output, "set2@example.org")
}

func TestSqlite(t *testing.T) {
	dir, err := os.MkdirTemp("", "pdscan")
	if err != nil {
		log.Fatal(err)
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

	checkSql(t, fmt.Sprintf("sqlite:%s", path))
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

func TestShowData(t *testing.T) {
	output := captureOutput(func() { Main("file://../testdata/email.txt", true, false, 10000, 1) })
	assert.Contains(t, output, "test@example.org")
}

// TODO fix
// func TestSampleSize(t *testing.T) {
// 	output := captureOutput(func() { Main("sqlite:../testdata/test.sqlite3", false, false, 250, 1) })
// 	assert.Contains(t, output, "sampling 250 rows from each")
// }

// helpers

func assertMatchName(t *testing.T, ruleName string, columnName string) {
	assertMatchNames(t, ruleName, []string{columnName})
}

func assertMatchNames(t *testing.T, ruleName string, columnNames []string) {
	columnValues := make([][]string, len(columnNames))
	for i := range columnValues {
		columnValues[i] = []string{}
	}
	assertMatch(t, ruleName, columnNames, columnValues)
}

func assertMatchValues(t *testing.T, ruleName string, values []string) {
	assertMatch(t, ruleName, []string{"col"}, [][]string{values})
}

func refuteMatchValues(t *testing.T, values []string) {
	matches := checkTableData(table{Name: "users"}, []string{"col"}, [][]string{values})
	assert.Equal(t, 0, len(matches))
}

func assertMatch(t *testing.T, ruleName string, columnNames []string, columnValues [][]string) {
	matches := checkTableData(table{Name: "users"}, columnNames, columnValues)
	assert.Equal(t, 1, len(matches))
	assert.Equal(t, ruleName, matches[0].RuleName)
}

func captureOutput(f func()) string {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = stdout
	return string(out)
}

func fileOutput(filename string) string {
	urlStr := fmt.Sprintf("file://../testdata/%s", filename)
	return captureOutput(func() { Main(urlStr, false, false, 10000, 1) })
}

func checkFile(t *testing.T, filename string, found bool) {
	output := fileOutput(filename)
	assert.Contains(t, output, "Found 1 file to scan...")
	if found {
		assert.Contains(t, output, fmt.Sprintf("%s:", filename))
	} else {
		assert.Contains(t, output, "No sensitive data found")
	}
}

func setupDb(driver string, dsn string) *sqlx.DB {
	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		log.Fatalln(err)
	}
	db.MustExec("DROP TABLE IF EXISTS users")
	return db
}

func checkSql(t *testing.T, urlStr string) string {
	output := captureOutput(func() { Main(urlStr, true, false, 10000, 1) })
	assert.Contains(t, output, "sampling 10000 rows")
	assert.NotContains(t, output, "users.id:")
	assert.Contains(t, output, "users.email:")
	assert.Contains(t, output, "users.phone:")
	assert.Contains(t, output, "users.street:")
	assert.Contains(t, output, "users.zip_code:")
	assert.Contains(t, output, "users.birthday:")
	assert.Contains(t, output, "users.ip:")
	assert.Contains(t, output, "users.ip2:")
	assert.Contains(t, output, "users.latitude+longitude:")
	assert.Contains(t, output, "users.access_token:")
	assert.Contains(t, output, "ITEMS.EMAIL:")
	assert.Contains(t, output, "ITEMS.ZipCode:")
	return output
}

func checkDocument(t *testing.T, urlStr string) {
	output := captureOutput(func() { Main(urlStr, true, false, 10000, 1) })
	assert.Contains(t, output, "sampling 10000 documents")
	assert.NotContains(t, output, "users._id:")
	assert.Contains(t, output, "users.email:")
	assert.Contains(t, output, "users.phone:")
	assert.Contains(t, output, "users.street:")
	assert.Contains(t, output, "users.zip_code:")
	assert.Contains(t, output, "users.birthday:")
	assert.Contains(t, output, "users.ip:")
	assert.Contains(t, output, "users.ip2:")
	assert.Contains(t, output, "users.latitude+longitude:")
	assert.Contains(t, output, "users.access_token:")

	// arrays
	assert.Contains(t, output, "users.emails:")
	assert.Contains(t, output, "first@example.org")
	assert.Contains(t, output, "second@example.org")

	// nested
	assert.Contains(t, output, "users.nested.email:")
	assert.Contains(t, output, "users.nested.zip_code:")
}
