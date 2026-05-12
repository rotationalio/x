# vault/v1

## Introduction

This package is for **encrypting application secrets before they hit your database**. You keep using whatever storage layer you already have; the vault hands it **opaque ciphertext** and remembers enough metadata on each row that only someone with your **long-term X25519 private key** can open it again.

You choose how rows are named (`Identifier`) and where bytes live (`Storage`). The core API works on **plaintext `[]byte`**—that is the shape you implement against when you wire things up. Optional helpers sit on top if you prefer strings or JSON in your app code.

Namespaces group rows: you pass a namespace string on every call. Each sealed blob is bound to that namespace; use the same one when you read or update, or call `MoveNamespace` when you relocate a secret. If the namespace does not match what was sealed, open fails (for example [`ErrNamespaceMismatch`](./errors/errors.go)).

---

## Key pieces

These pieces work together; you do not need every subpackage in every binary, but it helps to know what is what.

| Piece | Package | What it is |
|--------|---------|------------|
| **Vault** | [`v1`](./vault.go) | [`Vault`](./vault.go) interface + [`New`](./vault.go): seal and open rows with your key, `Storage`, and `Identifier`. |
| **Keys** | [`keys`](./keys/keys.go) | Optional Argon2id stretching ([`Derive`](./keys/keys.go)), random salt ([`RandSalt`](./keys/keys.go)), and mapping a 32-byte seed to an X25519 key ([`FromSeed`](./keys/keys.go)). |
| **Storage** | [`storage`](./storage/) | [`Storage`](./storage.go) interface; [`MemStorage`](./storage/mem.go) for tests and small tools. |
| **Identifier** | [`identifier`](./identifier/) | [`Identifier`](./identifier.go) interface; [`HexIdentifier`](./identifier/hex.go) is a small built-in example. |
| **Sentinels** | [`errors`](./errors/errors.go) | Stable errors for [`errors.Is`](https://pkg.go.dev/errors#Is). |
| **Wrappers** | [`stringvault`](./stringvault/), [`jsonvault`](./jsonvault/) | Same *method names* as the byte vault, different argument types (see below). |
| **Tests** | [`vaulttest`](./vaulttest/) | In-memory plaintext [`TestVault`](./vaulttest/vault.go) and contract tests for `Storage` / `Identifier`. |
| **Wire limits** | [`constants`](./constants/constants.go) | Sizes, magic bytes, and version constants used when building metadata. |

Lower-level wire and crypto helpers ([`models`](./models/), [`gcm`](./gcm/), [`suite`](./suite/suite.go)) are mainly for reading the format or extending the implementation; most callers only touch the table above.

---

## Initializing a vault

You need a **non-nil X25519 private key** (`*ecdh.PrivateKey`), a [`Storage`](./storage.go), and an [`Identifier`](./identifier.go). [`New`](./vault.go) validates inputs and returns [`(Vault, error)`](./vault.go).

**From a password.** Stretch with Argon2id, map to 32 bytes, then to a private key. Persist the **salt** wherever you will derive again (for example with the user or device record)—not inside each vault row. Tune Argon2 with [`keys.Params`](./keys/keys.go): [`DefaultParams`](./keys/keys.go) follows RFC 9106’s first recommendation (heavy memory); [`MemoryConstrainedParams`](./keys/keys.go) is the second profile. Clear sensitive slices with [`keys.Zero`](./keys/keys.go).

```go
import (
	v1 "go.rtnl.ai/x/vault/v1"
	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/keys"
	"go.rtnl.ai/x/vault/v1/storage"
)

func buildVault(password, salt []byte) (v1.Vault, error) {
	seed, err := keys.Derive(password, salt, keys.MemoryConstrainedParams(), keys.DerivedKeyBytes)
	if err != nil {
		return nil, err
	}
	defer keys.Zero(seed)

	priv, err := keys.FromSeed(seed)
	if err != nil {
		return nil, err
	}
	return v1.New(priv, storage.NewMemStorage(), identifier.HexIdentifier{})
}
```

**From 32 raw bytes** (KMS, HSM, or existing seed): [`ecdh.X25519().NewPrivateKey`](https://pkg.go.dev/crypto/ecdh#Curve.NewPrivateKey)(seed).

**From tests** without crypto, use [`vaulttest.NewTestVault`](./vaulttest/vault.go) (covered [below](#test-double-vaulttest)); it still implements [`Vault`](./vault.go) so wrappers and contracts behave the same.

This section stops at a real `v1.Vault` from [`New`](./vault.go). String and JSON layers are [later](#string-and-json-wrappers).

---

## Using the vault (bytes)

[`Vault`](./vault.go) is an interface: `Store`, `Retrieve`, `Update`, `CompareAndSwap`, `MoveNamespace`, `Delete`. All plaintext is **`[]byte`**. The namespace is an argument on each call, not a field on the vault value.

```go
import (
	"context"

	v1 "go.rtnl.ai/x/vault/v1"
)

func demo(ctx context.Context, v v1.Vault) error {
	id, err := v.Store(ctx, "my-app", []byte("secret"))
	if err != nil {
		return err
	}

	plain, err := v.Retrieve(ctx, "my-app", id)
	if err != nil {
		return err
	}
	_ = plain // []byte("secret")

	if err := v.Update(ctx, "my-app", id, []byte("v2")); err != nil {
		return err
	}
	if err := v.CompareAndSwap(ctx, "my-app", id, []byte("v2"), []byte("v2!")); err != nil {
		return err
	}
	if err := v.MoveNamespace(ctx, "my-app", "archived", id); err != nil {
		return err
	}
	return v.Delete(ctx, "archived", id)
}
```

`CompareAndSwap` decrypts the current row, compares plaintext to `currentPlain`, then re-seals `newPlain` only if they match—otherwise [`ErrWrongCurrent`](./errors/errors.go). Under the hood it relies on [`Storage.CompareAndSwap`](./storage.go) for atomic compare-and-swap on ciphertext.

Keep one vault (one wrapping key + storage wiring) for its lifetime; construct [`New`](./vault.go) again if you rotate the long-term key.

---

## String and JSON wrappers

[`stringvault`](./stringvault/) and [`jsonvault`](./jsonvault/) wrap a **non-nil** [`Vault`](./vault.go) you already built with [`New`](./vault.go). They **embed** it as field `Vault`, so `MoveNamespace` and `Delete` are promoted; use `w.Vault` when you need the raw-byte API in tests.

They are **separate types**: they do **not** implement `v1.Vault` (different method signatures), so you cannot pass them where a `v1.Vault` is required.

- **String** — Plaintext is UTF-8 `string`. Invalid UTF-8 on store or after decrypt → [`ErrInvalidUTF8`](./errors/errors.go).
- **JSON** — `Store` / `Update` take [`any`](https://pkg.go.dev/builtin#any) (marshaled with `encoding/json`). `Retrieve` unmarshals into a **non-nil** pointer. `CompareAndSwap` compares and swaps **JSON bytes** (`[]byte`); non-empty slices must be valid JSON. [`EqualJSON`](./jsonvault/jsonvault.go) compares two values by canonical marshaled bytes.

```go
import (
	"context"

	v1 "go.rtnl.ai/x/vault/v1"
	"go.rtnl.ai/x/vault/v1/jsonvault"
	"go.rtnl.ai/x/vault/v1/stringvault"
)

func wrap(ctx context.Context, v v1.Vault) error {
	s := stringvault.New(v)
sid, _ := s.Store(ctx, "ns", "hello")
_, _ = s.Retrieve(ctx, "ns", sid)

jw := jsonvault.New(v)
type payload struct{ N int `json:"n"` }
jid, _ := jw.Store(ctx, "ns", payload{N: 1})
var got payload
_ = jw.Retrieve(ctx, "ns", jid, &got)
```

---

## Storage: implementations and testing

[`Storage`](./storage.go) persists **opaque ciphertext** only. Keys are `(namespace, id)` strings; the vault never parses ciphertext. Treat blobs as immutable bytes: do not transform them in the driver.

**Behavior callers rely on**

| Method | Contract |
|--------|----------|
| `Create` | Insert-only. Duplicate `(namespace, id)` → error; use [`ErrDuplicateKey`](./errors/errors.go) when that is the case so [`errors.Is`](https://pkg.go.dev/errors#Is) works. |
| `Get` | Return stored bytes or [`ErrNotFound`](./errors/errors.go). Return a **copy** (or equivalent) so callers cannot mutate your buffer. |
| `Replace` | Overwrite an existing row; missing row → [`ErrNotFound`](./errors/errors.go). |
| `Delete` | Remove if present; **missing row must still return `nil`** (idempotent). |
| `CompareAndSwap` | Set `newCiphertext` only when the stored value **byte-equals** `oldCiphertext`. Wrong old value → [`ErrCASFailed`](./errors/errors.go). Missing row → [`ErrNotFound`](./errors/errors.go). |

Map SQL/driver “duplicate” and “no row” errors to the sentinels above where you can, so vault errors stay classifyable.

**Testing**

- In-process and unit tests: [`storage.NewMemStorage`](./storage/mem.go).
- Full contract: [`vaulttest.StorageConforms`](./vaulttest/storage.go) with your [`Identifier`](#identifier-implementations-and-testing) and a factory that returns a **fresh** `Storage` per subtest so cases do not share state.
- Targeted checks: exported [`CheckStorage…`](./vaulttest/storage.go) helpers return `error` for one scenario at a time.

---

## Identifier: implementations and testing

[`Identifier`](./identifier.go) separates **minting** ids (`New`, used from `Store`) from **validating** caller-supplied ids (`Parse`, used before any storage read/write). [`MarshalBinary`](./identifier.go) / [`UnmarshalBinary`](./identifier.go) map the canonical string id to opaque key bytes if your backend prefers binary primary keys; round-trip must recover the exact string.

**Expectations**

- `New` must return high-entropy ids; the vault does not check uniqueness before `Create`. For a given implementation, every successful `New` should use one **canonical encoded length** (so `Parse` can reject wrong lengths without touching storage).
- `Parse` must reject malformed ids, including any length that `New` would never produce. Every id previously returned from `New` for that type must parse.
- On `Store`, the vault calls `New()` then encrypts and `Create`s. On `Retrieve`, `Update`, `CompareAndSwap`, `MoveNamespace`, and `Delete`, it calls `Parse(id)` first.

[`identifier.HexIdentifier`](./identifier/hex.go) is a small stdlib-only example (random 16 bytes → 32 hex characters).

**Testing**

- [`vaulttest.IdentifierConforms`](./vaulttest/identifier.go) exercises many unique ids, fixed length, wrong-length rejection, and marshal round-trip. Distinct-id stress uses [`DefaultIdentifierDistinctSamples`](./vaulttest/identifier.go) by default; pass the same value to [`CheckIdentifierNewManyDistinct`](./vaulttest/identifier.go) if you call it directly.
- Other [`CheckIdentifier…`](./vaulttest/identifier.go) helpers cover single rules.

---

## Errors

Stable values live in [`errors`](https://pkg.go.dev/go.rtnl.ai/x/vault/v1/errors) ([`errors/errors.go`](./errors/errors.go)), including errors surfaced from [`suite`](./suite/suite.go) and [`keys`](./keys/keys.go), so one import is enough for classification.

Use [`errors.Is`](https://pkg.go.dev/errors#Is) in application code. Envelope and inner crypto paths tend to return **plain sentinels** (avoid leaking probe strings to untrusted parties). Storage, identifier, and JSON paths often wrap the underlying failure with [`errors.Join`](https://pkg.go.dev/errors#Join)(sentinel, err) so you can still match [`ErrStorage`](./errors/errors.go), [`ErrInvalidIdentifier`](./errors/errors.go), [`ErrJSONMarshal`](./errors/errors.go), and similar while logging the driver or `encoding/json` cause. Avoid echoing raw backend or JSON errors to clients.

---

## Test double: `vaulttest`

[`vaulttest.TestVault`](./vaulttest/vault.go) stores **plaintext bytes** through your [`Storage`](./storage.go) with **no envelope**. It implements [`Vault`](./vault.go), so you can embed it under [`stringvault`](./stringvault/) or [`jsonvault`](./jsonvault/) to test app logic without crypto cost, or to assert your `Storage` / `Identifier` implementations with the contract helpers above.

```go
import (
	"testing"

	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/storage"
	"go.rtnl.ai/x/vault/v1/vaulttest"
)

func TestWithMock(t *testing.T) {
	v := vaulttest.NewTestVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
	_ = v // stringvault.New(v), jsonvault.New(v), or StorageConforms / IdentifierConforms
}
```

[`NewTestVault`](./vaulttest/vault.go) requires a non-nil [`testing.TB`](https://pkg.go.dev/testing#TB) so it can fail tests on misuse.

---

## What else to know

Ideas that did not need their own top-level section but are easy to overlook:

- **Golden / version tests** — Package tests include fixed ciphertext fixtures; if you change the wire format, run/update those tests intentionally.
- **Contributors** — Wire structs live under [`models`](./models/); AEAD layout under [`gcm`](./gcm/). [`suite`](./suite/suite.go) names the supported suite ids on rows.
- **Operations** — `Update` re-encrypts and replaces the row in one shot (no read of current plaintext in the public API beyond what `CompareAndSwap` does internally). `MoveNamespace` copies a row to a new namespace and deletes the old one; partial failure can surface [`ErrMoveNamespaceIncomplete`](./errors/errors.go).
- **Security hygiene** — Protect the wrapping private key like any other long-term secret; treat vault errors as internal signals, not user-facing diagnostics.

If something is ambiguous, the [`Vault`](./vault.go) and [`Storage`](./storage.go) doc comments on the types are authoritative.
