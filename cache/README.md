# Cache

This package provides a few small generic caches:

- [`LRU`](#lru): a fixed-size least-recently-used cache with optional absolute expiration times.
- [`TTL`](#ttl): an unbounded cache whose values decide when they expire.
- [`SafeMap`](#safemap): a concurrency-safe map with per-key `GetOrCreate` coordination.

Import the package with:

```go
import "go.rtnl.ai/x/cache"
```

## LRU

Create an LRU with a maximum number of entries. The size must be greater than zero. The optional eviction callback is called whenever an entry is removed because of truncation, explicit eviction, or purge.

```go
lru, err := cache.NewLRU[string, User](128, func(key string, user User) {
	log.Printf("evicted %s", key)
})
if err != nil {
	return err
}

lru.Put("alice", user)

if user, ok := lru.Get("alice"); ok {
	use(user)
}
```

`Get` updates the access order, making the entry recently used. When the cache exceeds its configured size, the least recently used entry is evicted first. `Put` returns `true` when adding the value caused an eviction.

Use `Peek` to inspect an entry without changing its access order:

```go
user, expires, ok := lru.Peek("alice")
if ok {
	use(user)
	if !expires.IsZero() {
		log.Printf("expires at %s", expires)
	}
}
```

An entry can be given an absolute expiration time with `PutExpiry`. Expired entries are treated as misses by `Get` and `Contains`:

```go
expires := time.Now().Add(10 * time.Minute)
lru.PutExpiry("alice", user, expires)

if lru.Contains("alice") {
	// The entry exists and is not expired.
}
```

Other useful operations are:

- `Len()` returns the number of stored entries, including entries that have not yet been removed but are expired.
- `Evict(key)` removes one entry.
- `Purge()` removes every entry and invokes the eviction callback for each one.

## TTL

`TTL` stores values implementing `cache.Perishable`:

```go
type Perishable interface {
	Touch()
	Expired() bool
}
```

`Put` rejects already-expired values and calls `Touch` after storing a value. `Get` returns a miss for expired values and removes them. The cache is unbounded; the size passed to `NewTTL` only provides the initial map capacity.

```go
type Session struct {
	lastUsed time.Time
}

func (s *Session) Touch() {
	s.lastUsed = time.Now()
}

func (s *Session) Expired() bool {
	return time.Since(s.lastUsed) > 30*time.Minute
}

ttl := cache.NewTTL[string, *Session](256)
ttl.Put("alice", &Session{})

if session, ok := ttl.Get("alice"); ok {
	use(session)
}
```

Call `Clear` to remove currently expired values. For periodic cleanup, start the background cleanup routine with `Run` and stop it when the cache is no longer needed:

```go
if err := ttl.Run(time.Minute); err != nil {
	return err
}
defer ttl.Stop()
```

Only one cleanup routine can run at a time. Calling `Run` while one is active returns `cache.ErrAlreadyRunning`.

## SafeMap

`SafeMap` is a generic concurrency-safe map. Its zero value is ready to use:

```go
var users cache.SafeMap[string, *User]

users.Put("alice", user)
if user := users.Get("alice"); user != nil {
	use(user)
}

users.Delete("alice")
```

Use `Load` when you need to distinguish a missing key from a stored zero value:

```go
user, ok := users.Load("alice")
if ok {
	use(user)
}
```

`GetOrCreate` serializes creation for the same key while allowing creation for different keys to proceed independently. The callback is only used when the key has no committed value:

```go
user, err := users.GetOrCreate("alice", func(key string) (*User, error) {
	return loadUser(key)
})
if err != nil {
	return err
}
use(user)
```

If multiple goroutines call `GetOrCreate` for `"alice"` concurrently, they receive the same committed value and the create callback is serialized for that key. A callback error is returned and its value is not committed.

`FindKey` searches committed values and returns the first matching key:

```go
key, ok := users.FindKey(func(user *User) bool {
	return user.IsAdmin
})
if ok {
	log.Printf("admin user: %s", key)
}
```

`CloseAll` calls a callback for each committed entry and then removes the entries:

```go
users.CloseAll(func(key string, user *User) {
	user.Close()
})
```

All `SafeMap` operations can be used concurrently. Values themselves are not made thread-safe by the map, so shared mutable values still require their own synchronization.

## Testing

Run the package tests with:

```sh
go test ./cache
```

Run all repository tests with:

```sh
go test ./...
```
