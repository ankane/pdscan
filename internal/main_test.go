package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
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

func TestFileNoExt(t *testing.T) {
	checkFile(t, "email", true)
}

func TestFileTxt(t *testing.T) {
	checkFile(t, "email.txt", true)
}

func TestFileEmpty(t *testing.T) {
	checkFile(t, "empty.txt", false)
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
	db.MustExec("CREATE TABLE users (email text, email2 varchar(255), email3 char(255), latitude float, longitude float)")
	db.MustExec("INSERT INTO users (email, email2, email3) VALUES ('test@example.org', 'test@example.org', 'test@example.org')")

	urlStr := fmt.Sprintf("mysql://%s@localhost/pdscan_test", currentUser.Username)
	output := captureOutput(func() { Main(urlStr, false, false, 10000, 1) })
	assert.Contains(t, output, "Found 1 table to scan, sampling 10000 rows from each...")
	assert.Contains(t, output, "pdscan_test.users.email:")
	assert.Contains(t, output, "pdscan_test.users.email2:")
	assert.Contains(t, output, "pdscan_test.users.email3:")
	assert.Contains(t, output, "pdscan_test.users.latitude+longitude:")
}

func TestPostgres(t *testing.T) {
	db := setupDb("postgres", "dbname=pdscan_test sslmode=disable")
	db.MustExec("CREATE TABLE users (id serial, email text, email2 varchar(255), email3 char(255), ip inet, ip2 cidr, latitude float, longitude float)")
	db.MustExec("INSERT INTO users (email, email2, email3, ip, ip2) VALUES ('test@example.org', 'test@example.org', 'test@example.org', '127.0.0.1', '127.0.0.1')")

	output := captureOutput(func() { Main("postgres://localhost/pdscan_test?sslmode=disable", false, false, 10000, 1) })
	assert.Contains(t, output, "Found 1 table to scan, sampling 10000 rows from each...")
	assert.Contains(t, output, "public.users.email:")
	assert.Contains(t, output, "public.users.email2:")
	assert.Contains(t, output, "public.users.email3:")
	assert.Contains(t, output, "public.users.ip:")
	assert.Contains(t, output, "public.users.ip2:")
	assert.Contains(t, output, "public.users.latitude+longitude:")
}

func TestSqlite(t *testing.T) {
	output := captureOutput(func() { Main("sqlite:../testdata/test.sqlite3", false, false, 10000, 1) })
	assert.Contains(t, output, "Found 1 table to scan, sampling 10000 rows from each...")
	assert.Contains(t, output, "users.email:")
}

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

func checkFile(t *testing.T, filename string, found bool) {
	urlStr := fmt.Sprintf("file://../testdata/%s", filename)
	output := captureOutput(func() { Main(urlStr, false, false, 10000, 1) })
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
