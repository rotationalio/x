package dsn_test

import (
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/dsn"
)

func TestPGConnectionOptions(t *testing.T) {
	testCases := []struct {
		uri      string
		defaults map[string]string
		opts     *dsn.PostgresOptions
		connStr  string
		err      error
	}{
		{
			uri:      "postgresql://localhost:5432/mydb",
			defaults: nil,
			opts: &dsn.PostgresOptions{
				MaxIdleConns:    dsn.DefaultMaxIdleConns,
				MaxOpenConns:    dsn.DefaultMaxOpenConns,
				ConnMaxLifetime: dsn.DefaultConnMaxLifetime,
				ConnMaxIdleTime: dsn.DefaultConnMaxIdleTime,
			},
			connStr: "postgresql://localhost:5432/mydb",
			err:     nil,
		},
		{
			uri: "postgresql://localhost:5432/mydb?sslmode=disable",
			defaults: map[string]string{
				dsn.SSLMode:                 "prefer",
				dsn.ConnectTimeout:          "60",
				dsn.FallbackApplicationName: "myapp",
			},
			opts: &dsn.PostgresOptions{
				MaxIdleConns:    dsn.DefaultMaxIdleConns,
				MaxOpenConns:    dsn.DefaultMaxOpenConns,
				ConnMaxLifetime: dsn.DefaultConnMaxLifetime,
				ConnMaxIdleTime: dsn.DefaultConnMaxIdleTime,
			},
			connStr: "postgresql://localhost:5432/mydb?connect_timeout=60&fallback_application_name=myapp&sslmode=disable",
			err:     nil,
		},
		{
			uri: "postgresql://localhost:5432/mydb?sslmode=disable",
			defaults: map[string]string{
				dsn.SSLMode:                 "prefer",
				dsn.ConnectTimeout:          "60",
				dsn.FallbackApplicationName: "myapp",
				dsn.MaxIdleConns:            "128",
				dsn.MaxOpenConns:            "512",
				dsn.ConnMaxLifetime:         "60m",
				dsn.ConnMaxIdleTime:         "6m",
			},
			opts: &dsn.PostgresOptions{
				MaxIdleConns:    128,
				MaxOpenConns:    512,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 6 * time.Minute,
			},
			connStr: "postgresql://localhost:5432/mydb?connect_timeout=60&fallback_application_name=myapp&sslmode=disable",
		},
	}

	for i, tc := range testCases {
		uri, err := dsn.Parse(tc.uri)
		assert.Ok(t, err, "test case %d failed", i)

		connStr, opts, err := dsn.PGConnectionOptions(uri, tc.defaults)
		if tc.err != nil {
			assert.EqualError(t, err, tc.err.Error(), "test case %d failed", i)
			assert.Equal(t, "", connStr, "test case %d failed", i)
			assert.Nil(t, opts, "test case %d failed", i)
		} else {
			assert.Ok(t, err, "test case %d failed", i)
			assert.Equal(t, tc.connStr, connStr, "test case %d failed", i)
			assert.Equal(t, tc.opts, opts, "test case %d failed", i)
		}

	}
}
