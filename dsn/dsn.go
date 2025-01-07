package dsn

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrCannotParseDSN      = errors.New("could not parse dsn: missing provider or path")
	ErrCannotParseProvider = errors.New("could not parse dsn: incorrect provider")
	ErrCannotParsePort     = errors.New("could not parse dsn: invalid port number")
)

// DSN (data source name) contains information about how to connect to a database. It
// serves both as a mechanism to connect to the local database storage engine and as a
// mechanism to connect to the server databases from external clients. A typical DSN is:
//
// provider[+driver]://username[:password]@host:port/db?option1=value1&option2=value2
//
// This DSN provides connection to both server and embedded datbases. An embedded
// database DSN needs to specify relative vs absolute paths. Ensure an extra / is
// included for absolute paths to disambiguate the path and host portion.
type DSN struct {
	Provider string    // The provider indicates the database being connected to.
	Driver   string    // An additional component of the provider, separated by a + - it indicates what dirver to use.
	User     *UserInfo // The username and password (must be URL encoded for special chars)
	Host     string    // The hostname of the database to connect to.
	Port     uint16    // The port of the database to connect on.
	Path     string    // The path to the database (or the database name) including the directory.
	Options  Options   // Any additional connection options for the database.
}

// Contains user or machine login credentials.
type UserInfo struct {
	Username string
	Password string
}

// Additional options for establishing a database connection.
type Options map[string]string

func Parse(dsn string) (_ *DSN, err error) {
	var uri *url.URL
	if uri, err = url.Parse(dsn); err != nil {
		return nil, ErrCannotParseDSN
	}

	if uri.Scheme == "" || uri.Path == "" {
		return nil, ErrCannotParseDSN
	}

	d := &DSN{
		Host: uri.Hostname(),
		Path: strings.TrimPrefix(uri.Path, "/"),
	}

	scheme := strings.Split(uri.Scheme, "+")
	switch len(scheme) {
	case 1:
		d.Provider = scheme[0]
	case 2:
		d.Provider = scheme[0]
		d.Driver = scheme[1]
	default:
		return nil, ErrCannotParseProvider
	}

	if user := uri.User; user != nil {
		d.User = &UserInfo{
			Username: user.Username(),
		}
		d.User.Password, _ = user.Password()
	}

	if port := uri.Port(); port != "" {
		var pnum uint64
		if pnum, err = strconv.ParseUint(port, 10, 16); err != nil {
			return nil, ErrCannotParsePort
		}
		d.Port = uint16(pnum)
	}

	if params := uri.Query(); len(params) > 0 {
		d.Options = make(Options, len(params))
		for key := range params {
			d.Options[key] = params.Get(key)
		}
	}

	return d, nil
}

func (d *DSN) String() string {
	u := &url.URL{
		Scheme:   d.scheme(),
		User:     d.userinfo(),
		Host:     d.hostport(),
		Path:     d.Path,
		RawQuery: d.rawquery(),
	}

	if d.Host == "" {
		u.Path = "/" + d.Path
	}

	return u.String()
}

func (d *DSN) scheme() string {
	switch {
	case d.Provider != "" && d.Driver != "":
		return d.Provider + "+" + d.Driver
	case d.Provider != "":
		return d.Provider
	case d.Driver != "":
		return d.Driver
	default:
		return ""
	}
}

func (d *DSN) hostport() string {
	if d.Port != 0 {
		return fmt.Sprintf("%s:%d", d.Host, d.Port)
	}
	return d.Host
}

func (d *DSN) userinfo() *url.Userinfo {
	if d.User != nil {
		if d.User.Password != "" {
			return url.UserPassword(d.User.Username, d.User.Password)
		}
		if d.User.Username != "" {
			return url.User(d.User.Username)
		}
	}
	return nil
}

func (d *DSN) rawquery() string {
	if len(d.Options) > 0 {
		query := make(url.Values)
		for key, val := range d.Options {
			query.Add(key, val)
		}
		return query.Encode()
	}
	return ""
}
