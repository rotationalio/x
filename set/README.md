# Set

This package implements a `Set` and a `SyncSet` for common set operations. This is pretty standard data structure implementation that uses a `map[interface{}]struct{}` and also serves as a template for more specific set type implementations. The SyncSet provides thread-safe concurrent access to a set using locking.