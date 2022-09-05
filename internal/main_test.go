package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

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
			access_token text
		)
	`)
	db.MustExec("INSERT INTO users (email, phone, street, ip, ip2) VALUES ('test@example.org', '555-555-5555', '123 Main St', '127.0.0.1', '127.0.0.1')")

	checkSql(t, "postgres://localhost/pdscan_test?sslmode=disable")
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

	checkSql(t, fmt.Sprintf("sqlite:%s", path))
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

func checkSql(t *testing.T, urlStr string) {
	output := captureOutput(func() { Main(urlStr, false, false, 10000, 1) })
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
}
