# purser

Encrypt secrets before they reach your storage layer. `purser` separates persistence (`Hold`), crypto (`Locker`), and key routing (`Keyring`) so you can rotate keys and locker versions without losing old data.

See full API docs at [pkg.go.dev/go.rtnl.ai/x/purser](https://pkg.go.dev/go.rtnl.ai/x/purser).

## Install

```bash
go get go.rtnl.ai/x/purser
```

## Quick start

Import `go.rtnl.ai/x/purser`, describe your key with [`purser.NewPKCS8`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPKCS8), register it on a [`purser.Keyring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring), then use [`purser.Purser`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser) for row operations. Other key shapes ([`purser.NewPassword`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPassword), [`purser.NewSeed`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewSeed), [`purser.NewPrivateKey`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPrivateKey)) follow the same pattern—see **Key material** below.

```go
package main

import (
    "context"

    "go.rtnl.ai/x/purser"
)

func main() {
    // Create a context for the operations below.
    ctx := context.Background()

    // Load your PKCS#8-encoded X25519 private key bytes; typically from a KMS, file, or environment variable.
    pkcs8DER := loadPKCS8()

    // Set up an in-memory Hold for storing encrypted secrets.
    h, _ := purser.NewMemHold(purser.HexIdentifier{})

    // Set up an in-memory Keyring for managing crypto keys and routing.
    kr := purser.NewMemring()

    // Parse the PKCS#8 private key into a KeySpec, specifying the locker edition.
    spec, err := purser.NewPKCS8(pkcs8DER, purser.EditionV1)
    if err != nil {
        panic(err)
    }

    // Register the KeySpec, which builds a Locker and clears any sensitive key material from spec.
    lck, err := kr.Register(spec)
    if err != nil {
        panic(err)
    }

    // Set this Locker as the default for encrypting rows without a namespace binding.
    kr.SetDefault(lck)

    // Create a Purser, connecting it with the Hold (storage) and Keyring (crypto/routing).
    p, err := purser.New(h, kr)
    if err != nil {
        panic(err)
    }

    // Store an encrypted secret under the "app" namespace.
    res, err := p.Store(ctx, "app", []byte("hello"))
    if err != nil {
        panic(err)
    }
    _ = res.ID        // unique identifier of the sealed row
    _ = res.Namespace // namespace under which the row was stored
    _ = res.KeyID     // identifier of the key used for encryption
    _ = res.Edition   // locker edition/version used to seal the row

    // Retrieve and decrypt the secret using the Result from the Store op.
    plain, err := p.Retrieve(ctx, res.Namespace, res.ID)
    if err != nil {
        panic(err)
    }
    // 'plain' is now []byte("hello")

    // Update that secret in place to a new value.
    _, err = p.Update(ctx, res.Namespace, res.ID, []byte("world"))
    if err != nil {
        panic(err)
    }

    // Atomically swap when plaintext matches; returns Result like Store and Update.
    _, err := p.CompareAndSwap(ctx, res.Namespace, res.ID, []byte("world"), []byte("atomic update"))
    if err != nil {
        panic(err)
    }

    // Delete the secret by its ID.
    _ = p.Delete(ctx, "app", res.ID)
}

func loadPKCS8() []byte { /* ... */ return nil }
```

### Row operations

| Method | Returns | Notes |
|--------|---------|--------|
| [`Store`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.Store) | [`Result`](https://pkg.go.dev/go.rtnl.ai/x/purser#Result), `error` | New row; `Result.ID` is the new identifier |
| [`Update`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.Update) | `Result`, `error` | Re-seals in place |
| [`CompareAndSwap`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.CompareAndSwap) | `Result`, `error` | Swaps only when decrypted plaintext equals `currentPlain`; on error (including [`ErrWrongCurrent`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors#ErrWrongCurrent)) the row is unchanged and `Result` is zero |
| [`Retrieve`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.Retrieve) | `[]byte`, `error` | Decrypted plaintext |
| [`Delete`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.Delete) | `error` | Idempotent |
| [`MoveNamespace`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser.MoveNamespace) | `error` | Re-seals under a new namespace |

[`Result`](https://pkg.go.dev/go.rtnl.ai/x/purser#Result) fields: `ID`, `Namespace`, `KeyID`, `Edition` (non-secret routing metadata). Namespace-bound writes use [`Keyring.Bind`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring) when you need per-tenant lockers; otherwise use `SetDefault`. Register additional [`KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec) values before revoking old key ids when rotating.

### Key material ([`purser.KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec))

| Constructor | Use when |
|-------------|----------|
| [`purser.NewPassword`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPassword) | User password + persisted Argon2 salt ([`purser.RandSalt`](https://pkg.go.dev/go.rtnl.ai/x/purser#RandSalt) at enrollment) |
| [`purser.NewSeed`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewSeed) | You already derived a 32-byte seed |
| [`purser.NewPKCS8`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPKCS8) | PKCS#8-encoded private key bytes |
| [`purser.NewPrivateKey`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPrivateKey) | In-process `*ecdh.PrivateKey` (X25519 for v1) |

Each constructor returns a single-use [`purser.KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec). Pass it to `Register` on [`purser.Keyring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring) (recommended) or call `spec.Locker()` yourself and `SetDefault`. Sensitive fields are zeroed after `Locker()` runs; do not reuse a spec.

Every KeySpec constructor requires a non-empty edition (for example [`purser.EditionV1`](https://pkg.go.dev/go.rtnl.ai/x/purser#EditionV1)); an empty string returns [`ErrInvalidEdition`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors#ErrInvalidEdition).

## Root aliases (`aliases.go`)

| Category | Symbols |
|----------|---------|
| Editions | [`purser.EditionV1`](https://pkg.go.dev/go.rtnl.ai/x/purser#EditionV1) |
| Locker | [`purser.Locker`](https://pkg.go.dev/go.rtnl.ai/x/purser#Locker), [`purser.Sealer`](https://pkg.go.dev/go.rtnl.ai/x/purser#Sealer), [`purser.Labeler`](https://pkg.go.dev/go.rtnl.ai/x/purser#Labeler), [`purser.Keyer`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyer) |
| Keyring | [`purser.Keyring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring), [`purser.Registrator`](https://pkg.go.dev/go.rtnl.ai/x/purser#Registrator), [`purser.Namespacer`](https://pkg.go.dev/go.rtnl.ai/x/purser#Namespacer), [`purser.Router`](https://pkg.go.dev/go.rtnl.ai/x/purser#Router), [`purser.Memring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Memring), [`purser.NewMemring`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewMemring) |
| KeySpec | [`purser.KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec), [`purser.NewPassword`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPassword), [`purser.NewSeed`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewSeed), [`purser.NewPKCS8`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPKCS8), [`purser.NewPrivateKey`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPrivateKey) |
| Hold | [`purser.Hold`](https://pkg.go.dev/go.rtnl.ai/x/purser#Hold), [`purser.MemHold`](https://pkg.go.dev/go.rtnl.ai/x/purser#MemHold), [`purser.Identifier`](https://pkg.go.dev/go.rtnl.ai/x/purser#Identifier), [`purser.HexIdentifier`](https://pkg.go.dev/go.rtnl.ai/x/purser#HexIdentifier), [`purser.NewMemHold`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewMemHold) |
| KDF | [`purser.Params`](https://pkg.go.dev/go.rtnl.ai/x/purser#Params), [`purser.RandSalt`](https://pkg.go.dev/go.rtnl.ai/x/purser#RandSalt), [`purser.DefaultParams`](https://pkg.go.dev/go.rtnl.ai/x/purser#DefaultParams), [`purser.MemoryConstrainedParams`](https://pkg.go.dev/go.rtnl.ai/x/purser#MemoryConstrainedParams) |

Not aliased at the root (import subpackages): [`keyring/registry`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/registry) (edition dispatch and [`ParseKeyID`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/registry#ParseKeyID) for metadata-only routing), [`keyring/kdf`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/kdf) ([`Derive`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/kdf#Derive), [`SaltBytes`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/kdf#SaltBytes)), and [`purser/errors`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors) (sentinels for `errors.Is`). Prefer [`purser.KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec) over calling `registry.From*` directly.

Subpackages also cover conformance tests, wire layout, edition-specific crypto, and wrapper types (`holdtest`, `locker/v1`, `wrappers/json`, …).

## Security and operations

- **Built-in editions**: v1 wiring lives in [`keyring/registry`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/registry) (no `init` registration or blank imports); pass [`purser.EditionV1`](https://pkg.go.dev/go.rtnl.ai/x/purser#EditionV1) to [`purser.NewPassword`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewPassword), [`purser.NewSeed`](https://pkg.go.dev/go.rtnl.ai/x/purser#NewSeed), and related KeySpec constructors.
- **Key material**: build a [`purser.KeySpec`](https://pkg.go.dev/go.rtnl.ai/x/purser#KeySpec), then `Register` on [`purser.Keyring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring) (or `spec.Locker()` + `SetDefault` for a single key). Do not reuse specs after `Locker()` runs.
- **Salt handling**: store Argon2 salt with user/device metadata, not with each sealed row; generate with [`purser.RandSalt`](https://pkg.go.dev/go.rtnl.ai/x/purser#RandSalt).
- **Key routing**: [`purser.Keyring`](https://pkg.go.dev/go.rtnl.ai/x/purser#Keyring) `Route` parses wire (via [`registry.ParseKeyID`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/registry#ParseKeyID), then per-locker fallback in memring) and looks up the sealing key id.
- **Wire preamble**: sealed rows start with `wire.Magic` (`PURS`) plus a format-version byte. See `purser/wire`.
- **Metadata-only routing**: `registry.ParseKeyID(wire)` extracts the sealing key id without a keyring; decrypt still requires `Route` and a registered locker.
- **Default vs bind**: `LockerFor(namespace)` returns the bound locker, else the default; `ErrNoLocker` from `purser/errors` if neither is set.
- **Namespace binding**: ciphertext is namespace-bound; decrypting under the wrong namespace fails.
- **Compare-and-swap**: wrong plaintext returns [`ErrWrongCurrent`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors#ErrWrongCurrent) and a zero [`Result`](https://pkg.go.dev/go.rtnl.ai/x/purser#Result); hold-level CAS races surface [`ErrCASFailed`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors#ErrCASFailed).
- **Error checks**: classify with `errors.Is` using sentinels from [`purser/errors`](https://pkg.go.dev/go.rtnl.ai/x/purser/errors).
- **Memory hygiene**: use `memzero.Zero` on sensitive buffers you own; clear returned slices when done.

## Package map

```tree
purser/
├── purser.go                    Purser, New, row operations, Result
├── aliases.go                   purser.Locker, purser.Keyring, purser.EditionV1, … (see pkg.go.dev)
├── wire/                        Shared preamble magic (PURS) and offsets
├── errors/                      Sentinel errors (import as perrors)
├── hold/                        Hold interface + in-memory impl (purser.Hold, purser.NewMemHold)
│   ├── holdtest/                Conformance helpers (HoldConforms, Ciphertext)
│   └── identifier/              Identifier interface + hex (purser.HexIdentifier)
│       └── identifiertest/      Conformance helpers (IdentifierConforms)
├── internal/
│   ├── memzero/                 Byte-slice zeroing
│   └── nulllocker/              Null-encryption Locker for tests
├── keyring/
│   ├── keyring.go               Registrator, Namespacer, Router (purser.Keyring, …)
│   ├── keyspec/                 Primary key path (purser.NewPassword, …)
│   ├── kdf/                     Argon2id (purser.Params, purser.RandSalt, …)
│   ├── registry/                Edition dispatch behind KeySpec; ParseKeyID
│   ├── memring/                 In-memory Keyring (purser.NewMemring)
│   └── keyringtest/             Conformance helpers (KeyringConforms)
├── locker/
│   ├── locker.go                Sealer, Labeler, Keyer (purser.Locker)
│   ├── lockertest/              LockerConforms (version-neutral checks)
│   ├── lockertest/golden/       Golden tests for all locker versions
│   └── v1/                      X25519 + HKDF + AES-256-GCM
├── pursertest/                  NewTestPurser for app-level tests
├── benchmark/                   Hot-path benchmarks (optional `-bench` run)
└── wrappers/
    ├── json/                    JSON-typed Store/Update/CAS (CAS returns purser.Result)
    └── string/                  UTF-8 string Purser wrapper (CAS returns purser.Result)
```

## Testing

[`pursertest.NewTestPurser`](https://pkg.go.dev/go.rtnl.ai/x/purser/pursertest#NewTestPurser) builds a [`purser.Purser`](https://pkg.go.dev/go.rtnl.ai/x/purser#Purser) wired through Hold → Keyring → Locker with a null locker (fast orchestration tests). Use a real v1 locker when you need envelope crypto or `crypto/rand`.

Conformance suites (import the `*test` packages from `package foo_test`):

- `holdtest.HoldConforms(t, factory)` — nulllocker-shaped wire with extractable key id
- `holdtest.Ciphertext(tb, namespace, plaintext)` — seals `plaintext` with the same deterministic nulllocker fixture as `HoldConforms` (fixed seed, `VariantA`) and returns hold-ready wire bound to `namespace`; use when tests need pre-sealed blobs (seed a custom `Hold`, negative routing cases) without building a full Keyring/Purser stack. This is test-only wire: it is not v1 locker format, so [`registry.ParseKeyID`](https://pkg.go.dev/go.rtnl.ai/x/purser/keyring/registry#ParseKeyID) rejects it.
- `identifiertest.IdentifierConforms(t, idGen)`
- `keyringtest.KeyringConforms(t, newKeyring)`
- `lockertest.LockerConforms(factory)` — version-neutral `locker.Locker` invariants; returns `error`, use `assert` in tests

`purser_test.go` covers KeySpec registration, registry dispatch, purser orchestration, and multi-version routing.

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
- `FuzzParseKeyID` (`./purser/locker/v1`) — v1 `ParseKeyID` on envelope wire

```bash
go test -run '^$' -fuzz FuzzMeta_unmarshal -fuzztime 30s ./purser/locker/v1/models
go test -run '^$' -fuzz FuzzSealed_unmarshal -fuzztime 30s ./purser/locker/v1/models
go test -run '^$' -fuzz FuzzParseKeyID -fuzztime 30s ./purser/locker/v1
```

Commit inputs under `testdata/fuzz/FuzzXxx/` when a target finds a crasher.

### Benchmarks

Benchmarks are zero-cost under a normal `go test` run unless invoked with `-bench`. They live in [`benchmark/`](benchmark/) (`BenchmarkLocker`, `BenchmarkKeyring`, `BenchmarkRegistry`, `BenchmarkPurser`). You can take a snapshot JSON into `purser/benchmark/results` if you use `PURSER_BENCH_SNAPSHOT=1`; these will not be committed to the repo.

```bash
go test -run=^$ -bench=. -benchmem ./purser/benchmark
go test -run=^$ -bench=Locker/Seal -benchmem ./purser/benchmark
PURSER_BENCH_SNAPSHOT=1 go test -run=TestBenchmarkSnapshot -count=1 ./purser/benchmark  # ~1s; writes results/go*_*.json
python3 ./purser/benchmark/compare.py   # diff previous.json vs latest snapshot
```

Sample results at **256-byte** plaintext (`size=256`), **Apple M2**, `go 1.26.3` — other platforms will differ; `ns/op` varies with CPU load:

| Benchmark | ns/op | allocs/op |
|-----------|------:|----------:|
| `Locker/Seal/size=256` | ~74k | 31 |
| `Locker/Open/size=256` | ~40k | 25 |
| `Keyring/Route/1Locker/size=256` | ~142 | 1 |
| `Registry/ParseKeyID/size=256` | ~121 | 1 |
| `Purser/Store/size=256` | ~74k | 38 |
| `Purser/Retrieve/size=256` | ~38k | 27 |
| `Purser/Orchestration/Nulllocker/Store/size=256` | ~1.2k | 8 |

`Result.KeyID` and keyring routing return defensive copies on purpose; that shows up in orchestration benchmarks but keeps callers from mutating sealing keys in place.

## Development

### Add a new locker version (`vN`)

Copy `locker/v1/` layout. Example below uses `v2`.

#### Implement

- [ ] `locker/v2/locker.go` — `locker.Locker` (see [`locker/locker.go`](locker/locker.go))
- [ ] `locker/v2/models/` — seal/open wire types
- [ ] `locker/v2/keys.go` — `FromSeed`, `FromPassword`, `FromPKCS8`, `FromKey`
- [ ] `locker/v2/constants/` — `Edition`, `Version`, `Recipe`, `Context`
- [ ] Wire preamble: `wire.Magic` (`PURS`) + your `Version` byte (must be unique; v1 is `1`)

#### Register

- [ ] Add an `editionHooks` entry in [`keyring/registry/registry.go`](keyring/registry/registry.go)
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

`Keyring` composes `Registrator`, `Namespacer`, and `Router`:

- **Registrator**: `SetDefault` indexes the default locker; `Register`/`Revoke` add and remove lockers from the index (`Revoke` missing id → `ErrNoLocker`).
- **Namespacer**: `LockerFor` without default or bind → `ErrNoLocker`; namespace `Bind` overrides default; duplicate `Bind` → `ErrAlreadyBound`; `Unbind` returns the former locker (`ErrNotBound` when absent); `Namespaces` snapshot matches bindings.
- **Router**: `Route` and `ParseKeyID` resolve nulllocker test wire by key id; duplicate key id on index → `ErrDuplicateKeyID`.

#### `Locker` (`lockertest.LockerConforms`)

- Non-empty `KeyID` and defensive copy behavior.
- Seal/Open round-trip (including empty plaintext and empty namespace).
- Open rejects wrong namespace, empty wire, and zeroed garbage wire.
- `ParseKeyID` on self-sealed wire matches `KeyID`; empty input rejected.
