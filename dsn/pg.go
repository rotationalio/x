package dsn

import "time"

const (
	MaxIdleConns            = "max_idle_conns"
	MaxOpenConns            = "max_open_conns"
	ConnMaxLifetime         = "conn_max_lifetime"
	ConnMaxIdleTime         = "conn_max_idle_time"
	SSLMode                 = "sslmode"
	ConnectTimeout          = "connect_timeout"
	FallbackApplicationName = "fallback_application_name"

	DefaultMaxIdleConns    = 8
	DefaultMaxOpenConns    = 16
	DefaultConnMaxLifetime = 1 * time.Hour
	DefaultConnMaxIdleTime = 30 * time.Minute
)

type PostgresOptions struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Returns Postgres specific connection options for the given DSN as well as a new
// connection DSN string with the options removed so they are not duplicated. This
// function is used for providing specific options to a database connection manager.
// Defaults can be specified to provide default values for the connection options if
// they are not set in the DSN.
func PGConnectionOptions(uri *DSN, defaults map[string]string) (connStr string, opts *PostgresOptions, err error) {
	// Make a copy of the database URL to avoid modifying the original.
	dsn := uri.Clone()
	updateOptions(defaults, dsn.Options)

	opts = &PostgresOptions{
		MaxIdleConns:    DefaultMaxIdleConns,
		MaxOpenConns:    DefaultMaxOpenConns,
		ConnMaxLifetime: DefaultConnMaxLifetime,
		ConnMaxIdleTime: DefaultConnMaxIdleTime,
	}

	if maxIdleConns, err := dsn.Options.Int64(MaxIdleConns); err == nil {
		opts.MaxIdleConns = int(maxIdleConns)
		delete(dsn.Options, MaxIdleConns)
	}

	if maxOpenConns, err := dsn.Options.Int64(MaxOpenConns); err == nil {
		opts.MaxOpenConns = int(maxOpenConns)
		delete(dsn.Options, MaxOpenConns)
	}

	if connMaxLifetime, err := dsn.Options.Duration(ConnMaxLifetime); err == nil {
		opts.ConnMaxLifetime = connMaxLifetime
		delete(dsn.Options, ConnMaxLifetime)
	}

	if connMaxIdleTime, err := dsn.Options.Duration(ConnMaxIdleTime); err == nil {
		opts.ConnMaxIdleTime = connMaxIdleTime
		delete(dsn.Options, ConnMaxIdleTime)
	}

	return dsn.String(), opts, nil
}

// Updates the options map with the default values if the keys are not already set.
func updateOptions(defaults, options map[string]string) map[string]string {
	if options == nil {
		options = make(map[string]string, len(defaults))
	}

	for key, val := range defaults {
		if _, ok := options[key]; !ok {
			options[key] = val
		}
	}

	return options
}
