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
    "fmt"

    "go.rtnl.ai/x/purser"
    "go.rtnl.ai/x/purser/contract"
    "go.rtnl.ai/x/purser/hold"
    hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
    "go.rtnl.ai/x/purser/keyring"
    "go.rtnl.ai/x/purser/keyring/memring"
    "go.rtnl.ai/x/purser/registry"
    constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

func buildPurser(password, salt []byte) (contract.Purser, error) {
    // Derive a v1 locker from password+salt using Argon2id parameters.
    // Persist the salt with your user/device record.
    lck, err := registry.FromPassword(constv1.Edition, password, salt, keyring.MemoryConstrainedParams())
    if err != nil {
        return nil, fmt.Errorf("build locker: %w", err)
    }

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

Alternatively, construct the locker directly: `lockerv1.FromPassword(...)` returns `contract.Locker` without going through `registry`.

## Security and operations

- **Locker registration**: `registry.FromSeed`, `FromPassword`, `FromPKCS8`, `FromKey`, and `registry.ParseKeyID` dispatch over editions registered by each `locker/vN` in `init`. Importing `go.rtnl.ai/x/purser` links v1; add `installv2.go` (or similar) when you ship v2.
- **Salt handling**: store Argon2 salt with user/device metadata, not with each sealed row.
- **Key routing**: decrypt uses `Keyring.RouteKeyID`, so old rows remain readable when new lockers are added.
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
├── purser.go                    purser.New and row operations
├── installv1.go                 Links locker/v1 into the binary
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

1. Create `locker/vN/` implementing `contract.Locker` (`KeyID`, `Seal`, `Open`, `ParseKeyID`).
2. Add `locker/vN/constants` with `Edition`, wire `Version`, `Recipe`, and `Context`.
3. Add constructors and `register.go` calling `registry.Register(constants.Edition, registry.Hooks{…})` (see `locker/v1/register.go`). Do not import `purser` root from `locker/vN` (use `contract` + `registry` only).
4. Add `installvN.go` on the `purser` package: `import _ "go.rtnl.ai/x/purser/locker/vN"`.
5. Update `TestRegisteredLockerEditions` in `purser_test.go`: add `constvN.Edition` to `required`.
6. Add a `cases()` entry in `locker/lockertest/golden/golden_test.go` and `testdata/vN.dat`.
7. Run tests; `TestLockerImplementationsCovered` fails if a `locker/vN` with `locker.go` has no golden case.

`registry.FromPKCS8`, `FromKey`, and `ParseKeyID` try every registered edition automatically.

**Dispatch:** pass `constvN.Edition` to `registry.FromSeed` / `registry.FromPassword`, or call `lockervN.FromPassword` directly.

v1: `constants.Edition` (`v1`), wire `constants.Version` (byte `1`), `constants.Recipe`, `constants.Context` (HKDF).

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
