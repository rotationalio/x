package dsn_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/dsn"
)

func TestParse(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			uri      string
			expected *dsn.DSN
		}{
			{
				"sqlite3:///path/to/test.db",
				&dsn.DSN{Provider: "sqlite3", Path: "path/to/test.db"},
			},
			{
				"sqlite3:////absolute/path/test.db",
				&dsn.DSN{Provider: "sqlite3", Path: "/absolute/path/test.db"},
			},
			{
				"leveldb:///path/to/db",
				&dsn.DSN{Provider: "leveldb", Path: "path/to/db"},
			},
			{
				"leveldb:////absolute/path/db",
				&dsn.DSN{Provider: "leveldb", Path: "/absolute/path/db"},
			},
			{
				"postgresql://janedoe:mypassword@localhost:5432/mydb?schema=sample",
				&dsn.DSN{Provider: "postgresql", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 5432, Path: "mydb", Options: dsn.Options{"schema": "sample"}},
			},
			{
				"postgresql+psycopg2://janedoe:mypassword@localhost:5432/mydb?schema=sample",
				&dsn.DSN{Provider: "postgresql", Driver: "psycopg2", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 5432, Path: "mydb", Options: dsn.Options{"schema": "sample"}},
			},
			{
				"mysql://janedoe:mypassword@localhost:3306/mydb",
				&dsn.DSN{Provider: "mysql", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 3306, Path: "mydb"},
			},
			{
				"mysql+odbc://janedoe:mypassword@localhost:3306/mydb",
				&dsn.DSN{Provider: "mysql", Driver: "odbc", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 3306, Path: "mydb"},
			},
			{
				"cockroachdb+postgresql://janedoe:mypassword@localhost:26257/mydb?schema=public",
				&dsn.DSN{Provider: "cockroachdb", Driver: "postgresql", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 26257, Path: "mydb", Options: dsn.Options{"schema": "public"}},
			},
			{
				"mongodb+srv://root:password@cluster0.ab1cd.mongodb.net/myDatabase?retryWrites=true&w=majority",
				&dsn.DSN{Provider: "mongodb", Driver: "srv", User: &dsn.UserInfo{Username: "root", Password: "password"}, Host: "cluster0.ab1cd.mongodb.net", Path: "myDatabase", Options: dsn.Options{"retryWrites": "true", "w": "majority"}},
			},
		}

		for i, tc := range testCases {
			actual, err := dsn.Parse(tc.uri)
			assert.Ok(t, err, "test case %d failed", i)
			assert.Equal(t, tc.expected, actual, "test case %d failed", i)
		}

	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			uri string
			err error
		}{
			{"", dsn.ErrCannotParseDSN},
			{"sqlite3://", dsn.ErrCannotParseDSN},
			{"postgresql://jdoe:<mypassword>@localhost:foo/mydb", dsn.ErrCannotParseDSN},
			{"postgresql://localhost:foo/mydb", dsn.ErrCannotParseDSN},
			{"mysql+odbc+sand://jdoe:mypassword@localhost:3306/mydb", dsn.ErrCannotParseProvider},
			{"postgresql://jdoe:mypassword@localhost:656656/mydb", dsn.ErrCannotParsePort},
		}

		for i, tc := range testCases {
			_, err := dsn.Parse(tc.uri)
			assert.ErrorIs(t, err, tc.err, "test case %d failed", i)
		}
	})
}

func TestString(t *testing.T) {
	testCases := []struct {
		expected string
		uri      *dsn.DSN
	}{
		{
			"sqlite3:///path/to/test.db",
			&dsn.DSN{Provider: "sqlite3", Path: "path/to/test.db"},
		},
		{
			"sqlite3:////absolute/path/test.db",
			&dsn.DSN{Provider: "sqlite3", Path: "/absolute/path/test.db"},
		},
		{
			"leveldb:///path/to/db",
			&dsn.DSN{Provider: "leveldb", Path: "path/to/db"},
		},
		{
			"leveldb:////absolute/path/db",
			&dsn.DSN{Provider: "leveldb", Path: "/absolute/path/db"},
		},
		{
			"postgresql://localhost:5432/mydb?schema=sample",
			&dsn.DSN{Provider: "postgresql", Host: "localhost", Port: 5432, Path: "mydb", Options: dsn.Options{"schema": "sample"}},
		},
		{
			"postgresql+psycopg2://janedoe:mypassword@localhost:5432/mydb?schema=sample",
			&dsn.DSN{Provider: "postgresql", Driver: "psycopg2", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 5432, Path: "mydb", Options: dsn.Options{"schema": "sample"}},
		},
		{
			"mysql://janedoe:mypassword@localhost:3306/mydb",
			&dsn.DSN{Provider: "mysql", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 3306, Path: "mydb"},
		},
		{
			"mysql+odbc://janedoe@localhost:3306/mydb",
			&dsn.DSN{Provider: "mysql", Driver: "odbc", User: &dsn.UserInfo{Username: "janedoe"}, Host: "localhost", Port: 3306, Path: "mydb"},
		},
		{
			"cockroachdb+postgresql://janedoe:mypassword@localhost:26257/mydb?schema=public",
			&dsn.DSN{Provider: "cockroachdb", Driver: "postgresql", User: &dsn.UserInfo{Username: "janedoe", Password: "mypassword"}, Host: "localhost", Port: 26257, Path: "mydb", Options: dsn.Options{"schema": "public"}},
		},
		{
			"mongodb+srv://root:password@cluster0.ab1cd.mongodb.net/myDatabase?retryWrites=true&w=majority",
			&dsn.DSN{Provider: "mongodb", Driver: "srv", User: &dsn.UserInfo{Username: "root", Password: "password"}, Host: "cluster0.ab1cd.mongodb.net", Path: "myDatabase", Options: dsn.Options{"retryWrites": "true", "w": "majority"}},
		},
		{
			"cockroachdb://localhost:26257/mydb",
			&dsn.DSN{Driver: "cockroachdb", Host: "localhost", Port: 26257, Path: "mydb"},
		},
		{
			"//localhost:26257/mydb",
			&dsn.DSN{Host: "localhost", Port: 26257, Path: "mydb"},
		},
		{
			"/mydb",
			&dsn.DSN{Path: "mydb"},
		},
	}

	for i, tc := range testCases {
		actual := tc.uri.String()
		assert.Equal(t, tc.expected, actual, "test case %d failed", i)
	}
}
