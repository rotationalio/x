# DSN

A data source name (DSN) contains information about how to connect to a database in the form of a single URL string. This information makes it easy to provide connection information in a single data item without requiring multiple elements for the connection provider. This package provides parsing and handling of DSNs for database connections including both server and embedded database connections.

A typical DSN for a server is something like:

```
provider[+driver]://username[:password]@host:port/db?option1=value1&option2=value2
```

Whereas an embedded database usually just includes the provider and the path:

```
provider:///relative/path/to/file.db
provider:////absolute/path/to/file.db
```

Use the `dsn.Parse` method to parse this provider so that you can pass the connection details easily into your connection manager of choice.