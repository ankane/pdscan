package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestFile(t *testing.T) {
	Main("file://../testdata/email.txt", false, false, 10000, 1)
}

func TestFileEmpty(t *testing.T) {
	Main("file://../testdata/empty.txt", false, false, 10000, 1)
}

func TestFileTarGz(t *testing.T) {
	Main("file://../testdata/email.tar.gz", false, false, 10000, 1)
}

func TestFileZip(t *testing.T) {
	Main("file://../testdata/email.zip", false, false, 10000, 1)
}

func TestSqlite(t *testing.T) {
	Main("sqlite:../testdata/test.sqlite3", false, false, 10000, 1)
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
