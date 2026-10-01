# Set

Generic mathematical set types that replace `map[T]struct{}` with a small, readable API for membership and set algebra. `Set`/`SyncSet` elements use shallow (`==`) equality, so `T` must be `comparable`; `HashSet` elements use `Hash()` equality for complex types needing deep equality.

Import the package with:

```go
import "go.rtnl.ai/x/set"
```

All sets implement [`Container`](#container), so `Set`, `HashSet`, and `SyncSet` can be mixed in `Update`, `Union`, `Intersection`, and the other set operations (the element type must satisfy all involved sets' constraints).

- [`Set`](#set): single-threaded set backed by `map[T]struct{}`.
- [`HashSet`](#hashset): single-threaded set backed by `map[H]T`, equality by `Hash()`.
- [`SyncSet`](#syncset): concurrency-safe wrapper around `Set`, constructed with `NewSyncSet`/`MakeSyncSet`.
- [`Container`](#container): interface shared by every set in this package.

## Set

`Set[T comparable]` is not thread-safe; do not share it across goroutines without external synchronization. Use [`SyncSet`](#syncset) for concurrent use.

Create a set with `New` (from elements) or `Make` (with initial capacity):

```go
s := set.New(1, 2, 3)
t := set.Make[int](16)
t.Add(4, 5)
```

A zero-value `Set` is usable: `Add` lazily allocates the backing map. `Make` only pre-sizes the map; sets grow and shrink like maps.

Basic membership:

```go
modified := s.Add(4)        // true if the set changed
modified = s.Remove(1)      // true if the set changed
ok := s.Contains(2, 3)      // true only if ALL elements are present
n := s.Size()
empty := s.Empty()          // n == 0
s.Clear()                   // remove all elements
```

Iteration order is unordered. Use `Slice` for a snapshot or `Items` for `range`-over-func iteration (Go 1.23+):

```go
for v := range s.Items() {
    use(v)
}
vals := s.Slice() // []T, unordered
```

## HashSet

`HashSet[T Hashable[H], H Hash]` is not thread-safe; do not share it across goroutines without external synchronization. It implements [`Container[T]`](#container) with the same membership and algebra API as [`Set`](#set), but determines equality by hash instead of `==`, so it works for complex types needing deep equality.

Define `Hash()` on the element type. `H` must be a `Hash` (`~string` or an `int`/`uint` variant):

```go
type Color struct{ R, G, B uint8 }

func (c Color) Hash() string {
    return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}
```

Create a set with `NewHashSet` (from elements) or `MakeHashSet` (with initial capacity):

```go
s := set.NewHashSet(Color{255, 0, 0}, Color{0, 255, 0})
t := set.MakeHashSet[Color](16)
t.Add(Color{0, 0, 255})
```

Membership, mutation, iteration, and algebra match `Set`:

```go
modified := s.Add(Color{0, 0, 255}) // true if the hash was absent
ok := s.Contains(Color{255, 0, 0})  // hash lookup, not == comparison
for v := range s.Items() {
    use(v)
}
u := s.Union(t) // *set.HashSet[Color, string]
```

Three notes:

- Unlike `Set`, the zero value is not usable: `Add` calls the stored hash func, so always construct with `NewHashSet`/`MakeHashSet`. (`Clear` keeps the hash func, so a cleared set remains usable.)
- Hashes must uniquely identify values. Distinct values returning the same hash collide: `Add` keeps the first, `Contains`/`Remove` match by hash.
- `Union`, `Intersection`, `Difference`, and `SymmetricDifference` return a plain, non-thread-safe `*HashSet`; `Copy` returns an independent `*HashSet` as a `Container[T]`.

## SyncSet

`SyncSet[T comparable]` wraps `Set` with a `sync.RWMutex`.

```go
s := set.NewSyncSet("a", "b")
t := set.MakeSyncSet[string](16)
```

Reads (`Contains`, `Size`, `Empty`, `Slice`, `Items`) take a read lock; writes (`Add`, `Remove`, `Update`, `Clear`) take a write lock. Operations that take another `Container` (`Update`, `Equal`, `Disjoint`, `Subset`, `Superset`, `Union`, `Intersection`, `Difference`, `SymmetricDifference`) lock both sets only when the other operand is also a `*SyncSet[T]`:

```go
a := set.NewSyncSet(1, 2)
b := set.NewSyncSet(2, 3)
u := a.Union(b) // safe: both operands locked
```

Passing a plain `*Set[T]` (or any other `Container`) as the other operand is only safe if that container is not being mutated concurrently.

Two result-type notes:

- `Copy` on a `SyncSet` returns a new, independent `*SyncSet`.
- `Union`, `Intersection`, `Difference`, and `SymmetricDifference` on a `SyncSet` return a plain, non-thread-safe `*Set`, even when both operands are `*SyncSet`. Wrap or copy the result into a `SyncSet` if it will be shared.

## Container

`Container[T any]` is the interface every set implements:

```go
Add(...T) bool
Remove(...T) bool
Contains(...T) bool
Update(Container[T]) bool
Size() int
Empty() bool
Clear()
Copy() Container[T]
Disjoint(Container[T]) bool
Equal(Container[T]) bool
Subset(Container[T]) bool
Superset(Container[T]) bool
Union(Container[T]) Container[T]
Intersection(Container[T]) Container[T]
Difference(Container[T]) Container[T]
SymmetricDifference(Container[T]) Container[T]
Slice() []T
Items() iter.Seq[T]
```

`Add`, `Remove`, and `Update` report whether the receiver was modified, so re-adding an existing element returns `false`. `Contains` with multiple arguments is an AND: it returns `true` only if every argument is present.

Set algebra always allocates a new container and leaves both operands unchanged; `Update` and `Copy` are the in-place / cloning counterparts:

| Operation            | Meaning                                              |
|----------------------|------------------------------------------------------|
| `Union(o)`           | elements in either set                               |
| `Intersection(o)`    | elements in both sets                                |
| `Difference(o)`      | elements in the receiver but not in `o`              |
| `SymmetricDifference(o)` | elements in exactly one of the two sets          |
| `Equal(o)`           | same size and same elements                          |
| `Disjoint(o)`        | no elements in common                                |
| `Subset(o)`          | receiver is contained in `o`                         |
| `Superset(o)`        | receiver contains `o`                                |
| `Update(o)`          | add all elements of `o` into the receiver in place  |
| `Copy()`             | independent clone; mutating one does not affect the other |

```go
evens := set.New(2, 4, 6)
odds := set.New(1, 3, 5)

u := evens.Union(odds)               // {1,2,3,4,5,6}
i := evens.Intersection(set.New(4, 6, 8)) // {4,6}
d := evens.Difference(set.New(4, 6, 8))   // {2}
x := evens.SymmetricDifference(odds)      // {1,2,3,4,5,6}

evens.Update(odds) // evens is now {1,2,3,4,5,6}
clone := evens.Copy()
```

Because the algebra is defined on `Container`, mixed types work:

```go
plain := set.New(1, 2)
safe := set.NewSyncSet(2, 3)
u := plain.Union(safe) // *set.Set[int] with {1,2,3}
```