# purser

Encrypt secrets before they reach your storage layer. `purser` separates persistence (`Hold`), crypto (`Locker`), and key routing (`Keyring`) so you can rotate keys and locker versions without losing old data.

See full API docs at [pkg.go.dev/go.rtnl.ai/x/purser](https://pkg.go.dev/go.rtnl.ai/x/purser).

## Install

```bash
go get go.rtnl.ai/x/purser
```

## Quick start

Importing `purser` links locker v1 automatically. Use `registry` for edition-dispatched locker construction and `contract` for interface types.

```go
import (
    "context"
    "crypto/ecdh"
    "fmt"

    "go.rtnl.ai/x/purser"
    "go.rtnl.ai/x/purser/contract"
    "go.rtnl.ai/x/purser/hold"
    hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
    "go.rtnl.ai/x/purser/keyring"
    "go.rtnl.ai/x/purser/keyring/memring"
    lockerv1 "go.rtnl.ai/x/purser/locker/v1"
    constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
    "go.rtnl.ai/x/purser/registry"
)

// newPurser wires Hold + Keyring around an existing locker (any constructor below).
func newPurser(lck contract.Locker) (contract.Purser, error) {
    h, err := hold.NewMemHold(hexid.Identifier{})
    if err != nil {
        return nil, fmt.Errorf("build hold: %w", err)
    }
    kr, err := memring.New(lck)
    if err != nil {
        return nil, fmt.Errorf("build keyring: %w", err)
    }
    return purser.New(h, kr)
}

// registry.FromPassword — password + persisted salt, edition "v1".
func lockerFromPassword(password, salt []byte) (contract.Locker, error) {
    return registry.FromPassword(constv1.Edition, password, salt, keyring.MemoryConstrainedParams())
}

// registry.FromSeed — 32-byte seed (e.g. output of your own KDF), edition "v1".
func lockerFromSeed(seed []byte) (contract.Locker, error) {
    return registry.FromSeed(constv1.Edition, seed)
}

// registry.FromPKCS8 — PKCS#8 DER; tries each registered locker edition until one parses.
func lockerFromPKCS8(der []byte) (contract.Locker, error) {
    return registry.FromPKCS8(der)
}

// registry.FromKey — *ecdh.PrivateKey (X25519); tries each registered edition until one accepts it.
func lockerFromKey(priv *ecdh.PrivateKey) (contract.Locker, error) {
    return registry.FromKey(priv)
}

// Example: password path end-to-end.
func buildPurser(password, salt []byte) (contract.Purser, error) {
    lck, err := lockerFromPassword(password, salt)
    if err != nil {
        return nil, fmt.Errorf("build locker: %w", err)
    }
    return newPurser(lck)
}

// Same constructors without registry — call lockerv1 directly when you only need v1:
//   lockerv1.FromPassword(password, salt, keyring.MemoryConstrainedParams())
//   lockerv1.FromSeed(seed)
//   lockerv1.FromPKCS8(der)
//   lockerv1.FromKey(priv)

func exampleUsage(ctx context.Context, p contract.Purser) error {
    id, err := p.Store(ctx, "app", []byte("secret-v1"))
    if err != nil {
        return err
    }

    plain, err := p.Retrieve(ctx, "app", id)
    if err != nil {
        return err
    }
    fmt.Println(string(plain))

    if err := p.Update(ctx, "app", id, []byte("secret-v2")); err != nil {
        return err
    }

    if err := p.CompareAndSwap(ctx, "app", id, []byte("secret-v2"), []byte("secret-v3")); err != nil {
        return err
    }

    if err := p.MoveNamespace(ctx, "app", "archive", id); err != nil {
        return err
    }

    return p.Delete(ctx, "archive", id)
}
```

Use `registry` when you want edition dispatch (`constv1.Edition`); use `lockerv1.From*` when v1 is the only locker you link in.

## Security and operations

- **Locker registration**: each `locker/vN` calls `registry.Register` from `init` (duplicate editions return `ErrDuplicateLockerEdition`). Importing `go.rtnl.ai/x/purser` links registered editions via [`install.go`](install.go) blank imports.
- **Salt handling**: store Argon2 salt with user/device metadata, not with each sealed row.
- **Key routing**: `memring.RouteKeyID` calls `registry.ParseKeyID` then looks up the key id; unrecognized v1-class wire returns `ErrNoLocker`. Test-only wire (e.g. nulllocker) falls back to registered lockers in lexicographic key-id order.
- **Wire preamble**: all sealed rows start with `wire.Magic` (`PURS`) plus a format-version byte (`wire.VersionOffset`). See `purser/wire`.
- **Deterministic dispatch**: `registry.ParseKeyID` requires `PURS`, then routes by `Hooks.WireVersion`. `FromPKCS8` / `FromKey` try registered editions in sorted edition order.
- **Metadata-only routing**: `registry.ParseKeyID(wire)` extracts the sealing key id without a keyring; decrypt still requires `RouteKeyID` and a registered locker.
- **Active locker**: writes always use `Keyring.Active()`; rotate by registering a new locker and calling `SetActive`.
- **Namespace binding**: ciphertext is namespace-bound; decrypting under the wrong namespace fails.
- **Error checks**: classify with `errors.Is` using sentinels from `purser/errors`.
- **Memory hygiene**: use `memzero.Zero` on sensitive buffers you own; clear returned slices when done.

## Package map

```tree
purser/
├── contract/                    Purser, Locker, Keyring interfaces
├── registry/                    Edition registry; From* and ParseKeyID
├── wire/                        Shared preamble magic (PURS) and offsets
├── purser.go                    purser.New and row operations
├── install.go                   Blank-imports locker editions into the binary
├── errors/                      Shared sentinel errors (import as perrors)
├── hold/                        Hold interface + in-memory impl
│   ├── holdtest/                Conformance helpers (HoldConforms, Ciphertext)
│   └── identifier/              Identifier interface + hex/ impl
│       └── identifiertest/      Conformance helpers (IdentifierConforms)
├── internal/
│   ├── memzero/                 Byte-slice zeroing
│   └── nulllocker/              Null-encryption Locker for tests
├── keyring/                     Argon2id Derive, RandSalt, Params
│   ├── keyringtest/             Conformance helpers (KeyringConforms)
│   └── memring/                 In-memory Keyring impl
├── locker/
│   ├── lockertest/              LockerConforms (version-neutral locker checks)
│   ├── lockertest/golden/       Golden tests for all locker versions
│   └── v1/                      X25519 + HKDF + AES-256-GCM (constants, models, gcm, register.go)
├── pursertest/                  NewTestPurser for app-level tests
└── wrappers/
    ├── json/                    JSON-typed Purser wrapper
    └── string/                  UTF-8 string Purser wrapper
```

## Testing

`pursertest.NewTestPurser` builds a `contract.Purser` wired through Hold → Keyring → Locker with a null locker (fast orchestration tests). Use a real v1 locker when you need envelope crypto or `crypto/rand`.

Conformance suites (import the `*test` packages from `package foo_test`):

- `holdtest.HoldConforms(t, factory)` — nulllocker-shaped wire with extractable key id
- `holdtest.Ciphertext(tb, namespace, plaintext)` — same fixture for integration tests
- `identifiertest.IdentifierConforms(t, idGen)`
- `keyringtest.KeyringConforms(t, newKeyring)`
- `lockertest.LockerConforms(factory)` — version-neutral `contract.Locker` invariants; returns `error`, use `assert` in tests

`purser_test.go` covers registry dispatch, purser orchestration, and multi-version routing.

Run all package tests (from the `x` module root):

```bash
go test -count=1 ./purser/...
go test -race -count=1 ./purser/...
```

### Fuzz targets

Fuzz tests are zero-cost under a normal `go test` run unless invoked with `-fuzz`.

Available targets:

- `FuzzMeta_unmarshal` (`./purser/locker/v1/models`) — `models.Meta.UnmarshalBinary`
- `FuzzSealed_unmarshal` (`./purser/locker/v1/models`) — `models.Sealed.UnmarshalBinary`
- `FuzzParseKeyID` (`./purser/locker/v1`) — `contract.Locker.ParseKeyID` for v1 envelope

```bash
go test -run '^$' -fuzz FuzzMeta_unmarshal -fuzztime 30s ./purser/locker/v1/models
go test -run '^$' -fuzz FuzzSealed_unmarshal -fuzztime 30s ./purser/locker/v1/models
go test -run '^$' -fuzz FuzzParseKeyID -fuzztime 30s ./purser/locker/v1
```

Commit inputs under `testdata/fuzz/FuzzXxx/` when a target finds a crasher.

## Development

### Add a new locker version (`vN`)

Copy `locker/v1/` layout. Example below uses `v2`.

#### Implement

- [ ] `locker/v2/locker.go` — `contract.Locker` (see [`contract`](contract/contract.go))
- [ ] `locker/v2/models/` — seal/open wire types
- [ ] `locker/v2/keys.go` — `FromSeed`, `FromPassword`, `FromPKCS8`, `FromKey`
- [ ] `locker/v2/constants/` — `Edition`, `Version`, `Recipe`, `Context`
- [ ] Wire preamble: `wire.Magic` (`PURS`) + your `Version` byte (must be unique; v1 is `1`)

#### Register

- [ ] `locker/v2/register.go` — `registry.Register` in `init` (copy [`locker/v1/register.go`](locker/v1/register.go))
- [ ] `purser/install.go` — add `_ "go.rtnl.ai/x/purser/locker/v2"`
- [ ] Do **not** import `go.rtnl.ai/x/purser` from `locker/v2`

#### Test

- [ ] `lockertest.LockerConforms` in `locker/v2` tests
- [ ] `purser_test.go` — add `"v2"` to `TestRegisteredLockerEditions`
- [ ] `golden/golden_test.go` — `cases()` entry `version: "v2"` (must match folder name)
- [ ] `go test ./purser/...`

#### Quick reference (v1)

| Field | v1 value |
|-------|----------|
| `Edition` | `"v1"` |
| `Version` (wire byte) | `1` |
| Magic | `PURS` (shared, [`wire`](wire/wire.go)) |

### Golden fixtures

Fixtures live in `locker/lockertest/golden/testdata/*.dat` and auto-generate when missing.
Delete fixtures and rerun tests to regenerate.

### Conformance invariants

#### `Hold` (`holdtest.HoldConforms`)

- Create/Get round-trip preserves stored wire bytes (nulllocker-shaped blobs in tests; v1 in production).
- Duplicate `CreateWithIdentifier` returns `ErrDuplicateKey`.
- CAS succeeds only on correct `old`; mismatch returns `ErrCASFailed`.
- Missing row `Get`/`Replace`/CAS returns `ErrNotFound`.
- Delete is idempotent.
- Namespace isolation is enforced.
- Replace updates stored data.
- Wire includes an extractable key id compatible with Purser routing.

#### `Identifier` (`identifiertest.IdentifierConforms`)

- `New()` values are unique and stable-length.
- `Parse(New())` succeeds.
- Off-by-one length IDs are rejected.
- Marshal/unmarshal round-trip preserves canonical string form.
- `New()` and `MarshalBinary()` are non-empty.

#### `Keyring` (`keyringtest.KeyringConforms`)

- `New(nil, ...)` is rejected.
- `Active()` returns write locker.
- `Lookup` resolves registered key IDs and misses unknown IDs.
- `Register(nil)` is rejected; duplicates return `ErrDuplicateKeyID`.
- `SetActive(nil)` is rejected; valid `SetActive` switches active and keeps locker routable.
- `RouteKeyID` routes ciphertext sealed by a registered non-active locker (nulllocker wire in tests).
- Unroutable ciphertext returns `ErrNoLocker`.

#### `Locker` (`lockertest.LockerConforms`)

- Non-empty `KeyID` and defensive copy behavior.
- Seal/Open round-trip (including empty plaintext and empty namespace).
- Open rejects wrong namespace, empty wire, and zeroed garbage wire.
- `ParseKeyID` on self-sealed wire matches `KeyID`; empty input rejected.
