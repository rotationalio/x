# purser

Encrypt secrets before they reach your storage layer. `purser` separates persistence (`Hold`), crypto (`Locker`), and key routing (`Keyring`) so you can rotate keys and locker versions without losing old data.

See full API docs at [pkg.go.dev/go.rtnl.ai/x/purser](https://pkg.go.dev/go.rtnl.ai/x/purser).

## Install

```bash
go get go.rtnl.ai/x/purser
```

## Quick start

```go
import (
    "context"
    "fmt"

    "go.rtnl.ai/x/purser"
    "go.rtnl.ai/x/purser/hold"
    hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
    "go.rtnl.ai/x/purser/keyring"
    "go.rtnl.ai/x/purser/keyring/memring"
    v1 "go.rtnl.ai/x/purser/locker/v1"
)

func buildPurser(password, salt []byte) (purser.Purser, error) {
    // Derive a v1 locker from password+salt using Argon2id parameters.
    // Persist the salt with your user/device record.
    lck, err := v1.FromPassword(password, salt, keyring.MemoryConstrainedParams())
    if err != nil {
        return nil, fmt.Errorf("build locker: %w", err)
    }

    // In-memory Hold for quick start. Replace with your DB-backed Hold in production.
    h, err := hold.NewMemHold(hexid.Identifier{})
    if err != nil {
        return nil, fmt.Errorf("build hold: %w", err)
    }

    // Create a keyring with this locker as active (write key).
    // Register more lockers here when you add new versions or rotated keys.
    kr, err := memring.New(lck)
    if err != nil {
        return nil, fmt.Errorf("build keyring: %w", err)
    }

    // Compose Purser from Hold + Keyring.
    return purser.New(h, kr)
}

func exampleUsage(ctx context.Context, p purser.Purser) error {
    // Store returns an opaque identifier; plaintext is sealed before persistence.
    id, err := p.Store(ctx, "app", []byte("secret-v1"))
    if err != nil {
        return err
    }

    // Retrieve returns plaintext if namespace and identifier are valid.
    plain, err := p.Retrieve(ctx, "app", id)
    if err != nil {
        return err
    }
    fmt.Println(string(plain)) // "secret-v1"

    // Update re-seals with the active locker and replaces stored ciphertext.
    if err := p.Update(ctx, "app", id, []byte("secret-v2")); err != nil {
        return err
    }

    // CompareAndSwap updates only if current plaintext matches expected bytes.
    if err := p.CompareAndSwap(ctx, "app", id, []byte("secret-v2"), []byte("secret-v3")); err != nil {
        return err
    }

    // MoveNamespace decrypts in old namespace and re-seals in new namespace.
    if err := p.MoveNamespace(ctx, "app", "archive", id); err != nil {
        return err
    }

    // Delete is idempotent: deleting a missing row returns nil.
    return p.Delete(ctx, "archive", id)
}
```

## Security and operations

- **Salt handling**: store Argon2 salt with user/device metadata, not with each sealed row.
- **Key routing**: decrypt uses `RouteKeyID`, so old rows remain readable when new lockers are added.
- **Active locker**: writes always use `Keyring.Active()`; rotate by registering a new locker and calling `SetActive`.
- **Namespace binding**: ciphertext is namespace-bound; decrypting under the wrong namespace fails.
- **Error checks**: classify with `errors.Is` using sentinels from `purser/errors`.
- **Memory hygiene**: implementations use `purser.Zero` on sensitive buffers they own (for example intermediate plaintext or DEK material); clear returned slices yourself when you are done with them.

## Package map

```tree
purser/
├── purser.go, purserimpl.go     Purser, Locker, Keyring interfaces + New()
├── helpers.go                   Shared utilities (Zero)
├── errors/                      Shared sentinel errors (import as perrors)
├── hold/                        Hold interface + in-memory impl
│   ├── holdtest/                Conformance helpers (HoldConforms)
│   └── identifier/              Identifier interface + hex/ impl
│       └── identifiertest/      Conformance helpers (IdentifierConforms)
├── internal/nulllocker/         Null-encryption Locker for tests (not importable externally)
├── keyring/                     Argon2id Derive, RandSalt, Params
│   ├── keyringtest/             Conformance helpers (KeyringConforms)
│   └── memring/                 In-memory Keyring impl
├── locker/
│   ├── lockertest/              LockerConforms (version-neutral locker checks)
│   ├── lockertest/golden/       Golden tests for all locker versions
│   └── v1/                      X25519 + HKDF + AES-256-GCM envelope locker
├── pursertest/                  NewTestPurser for app-level tests
└── wrappers/
    ├── json/                    JSON-typed Purser wrapper
    └── string/                  UTF-8 string Purser wrapper
```

## Testing

`pursertest.NewTestPurser` builds a real `purser.Purser` wired through Hold → Keyring → Locker with a test-only null locker (fast orchestration tests without v1 crypto). Use a real v1 locker when you need to exercise `crypto/rand` or envelope behavior.

Conformance suites (import the `*test` packages from your `package foo_test` tests):

- `holdtest.HoldConforms(t, factory)`
- `identifiertest.IdentifierConforms(t, idGen)`
- `keyringtest.KeyringConforms(t, newKeyring)`
- `lockertest.LockerConforms(factory)` — version-neutral `Locker` invariants (used by v1 and null-locker tests)

Run all package tests (from the `x` module root):

```bash
go test -count=1 ./purser/...
go test -race -count=1 ./purser/...
```

## Development

### Add a new locker version (`vN`)

1. Create `locker/vN/` implementing `purser.Locker` (`KeyID`, `Seal`, `Open`, `ParseKeyID`).
2. Add `SeedBytes`, `FromSeed`, and `FromPassword` constructors for that version.
3. Add wire constants/models for the version.
4. Add a golden test case in `locker/lockertest/golden/golden_test.go`.
5. Run tests; coverage checks fail if a `vN` implementation is missing from golden coverage.

### Golden fixtures

Fixtures live in `locker/lockertest/golden/testdata/*.dat` and auto-generate when missing.
Delete fixtures and rerun tests to regenerate.

### Conformance invariants

#### `Hold` (`holdtest.HoldConforms`)

- Create/Get round-trip preserves ciphertext bytes.
- Duplicate `CreateWithIdentifier` returns `ErrDuplicateKey`.
- CAS succeeds only on correct `old`; mismatch returns `ErrCASFailed`.
- Missing row `Get`/`Replace`/CAS returns `ErrNotFound`.
- Delete is idempotent.
- Namespace isolation is enforced.
- Replace updates stored data.

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
- `RouteKeyID` can route non-active locker formats via parse fallback.
- Unroutable ciphertext returns `ErrNoLocker`.

#### `Locker` (`lockertest.LockerConforms`)

- Non-empty `KeyID` and defensive copy behavior.
- Seal/Open round-trip (including empty plaintext and empty namespace).
- Open rejects wrong namespace, empty wire, and zeroed garbage wire.
- `ParseKeyID` on self-sealed wire matches `KeyID`; empty input rejected.
