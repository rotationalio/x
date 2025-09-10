# Locks

The locks package implements a key-based lock mechanism that uses a crc32 hash to
distribute keys across a fixed number of locks. This allows for concurrent access
to different keys without contention, but fixes the number of locks (and therefore the
amount of available concurrency) to ensure that memory usage is bounded.

Locks help us implement simple transactions for the storage engine. At the start of a
transaction before any other operation is performed, the transaction must acquire
its read and write locks to all keys it wants to access in the transaction. The locks
are acquired in a consistent order to prevent deadlocks.

Reference: [Handling Deadlocks in Golang Gracefully](https://medium.com/@ksandeeptech07/handling-deadlocks-in-golang-gracefully-1f661c341a1d)

## Usage

Create a new key lock with a fixed number of locks. The greater `nlocks` is, the greater concurrency there is across the entire key space at the cost of more memory. I recommend allocating at least 1024 locks to ensure suitable performance.

```golang
mu := locks.New(1024)
```

You can acquire locks or read locks for a set of keys. Note that the same exact keyset should be unlocked all at once and multiple calls to lock in the same go routine should be avoided.

```golang
func Update(ids []ulid.ULID, value []byte) {
    keys := make([][]byte, 0, len(ids))
    for _, id := range ids {
        keys = append(keys, id[:])
    }

    mu.Lock(keys...)
    defer mu.Unlock(keys...)

}

func Fetch(ids []ulid.ULID) {
    keys := make([][]byte, 0, len(ids))
    for _, id := range ids {
        keys = append(keys, id[:])
    }

    mu.RLock(keys...)
    defer mu.RUnlock(keys...)
}
```

The lock function itself will sort the keys to ensure the keys are unlocked in the same order so no sorting is required from the user.