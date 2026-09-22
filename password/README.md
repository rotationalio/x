# Password

This package has two primary functionalities:

1. Assessing the strength of a password
2. Generating passwords with a password policy

It also includes a CLI program for quickly generating passwords on the command line.

This package is primarily used by our devops tools to generate passwords for deployments (e.g. superuser passwords, database passwords, etc). It is also used by our authentication tools to force users to create strong passwords with a specific password policy.

Basic usage:

```go
import "go.rtnl.ai/x/password"

// Score the strength of a password.
strength := password.Check("Xk7#mQp2$vLr9Tz")  // strong

// Generate a password using the default policy.
pw, err := password.Generate("")
```

## Getting Started with the CLI Program

Install the CLI program with Go:

```
$ go install go.rtnl.ai/x/password/cmd/mkpasswd@latest
```

The `mkpasswd` command should now be available in your `$PATH`. You can set a default policy by creating a policy JSON file in `~/.config/mkpasswd/policies.json`:

```json
{
    "default:" {
        "length": 14,
        "charsets": [
            "differentiable",
            "digits",
            "symbols",
        ],
        "define": {
            "symbols": "!@#$%^&*()_-+=.><,?"
        },
        "require": [
            "digits",
            "uppercase",
            "lowercase",
            "symbols"
        ],
        "urlencode": false
    }
}
```

You can also set other named policies in this file to use them if needed. If you need to set a different policies configuration file, set the `$PASSWORD_POLICIES` environment variable with the path to the policy JSON file. Environment variables in the path are expanded, so `$HOME/policies.json` works.

The available flags are:

| Flag                 | Description                                              |
|----------------------|----------------------------------------------------------|
| `-n <int>`           | Number of passwords to generate (default 1)              |
| `-p`, `-policy`      | Name of the policy to use (default `default`)            |
| `-N`                 | Do not output a newline after each password              |
| `-list`              | List the names of the available policies                 |
| `-show`              | Print the selected policy as JSON instead of a password  |
| `-config`            | Print the path to the policies configuration file        |

```
$ mkpasswd -n 3 -p basic
$ mkpasswd -show -p default
$ mkpasswd -config
```

## API

### Check

```go
func Check(password string, options ...CheckOption) Strength
```

`Check` scores a password with the [strength algorithm](#strength) and returns a `Strength` value between `Insecure` and `Durable`. It performs no policy validation — use [`Policy.Check`](#policycheck) for that.

```go
switch s := password.Check(pw); {
case s < password.Moderate:
    return errors.New("please choose a stronger password")
default:
    fmt.Println("password is", s) // e.g. "password is strong"
}
```

Two options are available to explain how a score was reached, which is useful for debugging a policy or giving feedback to a user:

| Option              | Description                                                          |
|---------------------|----------------------------------------------------------------------|
| `WithAnalyze()`     | Writes each scoring decision to `os.Stdout`                          |
| `WithWriter(w)`     | Writes each scoring decision to `w` (implies `WithAnalyze()`)        |

```go
password.Check("Xk7#mQp2$vLr9Tz", password.WithWriter(os.Stdout))
```

```
more than 8 characters strength is now weak
uppercase charset is present, strength is now soft
lowercase charset is present, strength is now moderate
numbers charset is present, strength is now hard
symbols charset is present, strength is now strong
```

`Strength` marshals to and from JSON as its lowercase name (e.g. `"moderate"`), so it can be used directly in API payloads and in the `strength` field of a policy.

### Generate

```go
func Generate(policyName string) (string, error)
```

`Generate` loads the named policy and generates a password that satisfies it. An empty name is equivalent to `"default"`. Policies are loaded from `~/.config/mkpasswd/policies.json` or from the path in `$PASSWORD_POLICIES`, falling back to the [built-in policies](#policies) when no configuration file exists.

```go
pw, err := password.Generate("")       // default (strong) policy
pw, err := password.Generate("basic")  // built-in basic policy
```

It returns an error if the policy name is unknown, if the configuration file cannot be read or parsed, or if generation fails (see [`Policy.Generate`](#policygenerate)). Because the configuration file can be changed by the operator, always handle the error rather than assuming the named policy exists.

### Policy.Check

```go
func (p *Policy) Check(password string) error
```

`Policy.Check` returns `nil` if the password satisfies the policy and an error describing the first unmet constraint otherwise. Constraints are evaluated in order:

1. If `urlencode` is set, the password is URL path-unescaped before any other check.
2. If `strength` is set above `insecure`, `Check(password)` must meet or exceed it.
3. If `length.min` is set, the password must be at least that many bytes.
4. If `length.max` is set **and** is greater than `length.min`, the password must be at most that many bytes. A fixed-length policy (`"length": 16`) therefore constrains generation but only enforces a minimum on check.
5. Every character set named in `require` must contribute at least one character.

```go
policy, err := password.Load("default")
if err != nil {
    return err
}

if err := policy.Check(userSuppliedPassword); err != nil {
    return fmt.Errorf("password rejected: %w", err)
}
```

Note that lengths are measured in bytes, not runes, which matters for passwords containing multi-byte characters.

### Policy.Generate

```go
func (p *Policy) Generate() (string, error)
```

`Policy.Generate` generates a password that satisfies the policy. It first validates that every character set named in `charsets` and `require` resolves to a non-empty set, returning an error immediately if one does not. It then makes up to 8 attempts to:

1. Choose a length from `length` (or 14 if no length is set).
2. Distribute that many characters across the `charsets` using their probabilities or integer weights.
3. Draw the characters for each set and shuffle the result with a Fisher-Yates shuffle.
4. Validate the candidate with `Policy.Check`, URL-encoding it if `urlencode` is set.

An attempt is discarded when the chosen length is zero or a required character set was allocated no characters. If all 8 attempts fail, it returns `ErrGenerationFailed`, which usually means the policy is unsatisfiable — most often because a name in `require` does not appear in `charsets` (see [Policies](#policies)), or because more distinct sets are required than the length allows.

```go
policy := &password.Policy{
    Length:   &password.Range{Min: 12, Max: 20},
    Charsets: []*password.CharSelect{{Name: "differentiable"}, {Name: "symbols"}},
    Require:  []string{"differentiable", "symbols"},
}

pw, err := policy.Generate()
if errors.Is(err, password.ErrGenerationFailed) {
    // policy could not be satisfied in 8 attempts
}
```

The password characters and the shuffle use `crypto/rand`. Note, however, the length within a `min`/`max` range and the distribution of characters across sets use `math/rand`.

## Policies

Password policies define how passwords are generated and can also be used to verify that a given password meets a specific policy (e.g. to use a different metric other than strength).

This package ships with two built-in policies: a basic policy and a stronger default policy. Either built-in can be replaced by defining a policy with the same name in the policies configuration file.

A policy is a JSON object with the following fields, all of which are optional:

| Field       | Type                | Description                                                                            |
|-------------|---------------------|------------------------------------------------------------------------------------------|
| `strength`  | string              | Minimum [strength](#strength) name (e.g. `"moderate"`) a password must score             |
| `length`    | int or `{min, max}` | Fixed length, or an inclusive range to choose from; defaults to 14                       |
| `charsets`  | array               | Character sets to draw from, as names or `{"name": ..., "prob": ...}` objects            |
| `define`    | object              | Overrides or new character sets, mapping a name to its characters                        |
| `require`   | array of strings    | Character sets that must each contribute at least one character                          |
| `urlencode` | bool                | URL path-encode generated passwords and decode them before checking                      |

The `prob` values in `charsets` are interpreted one of two ways. If they sum to 1.0 they are treated as probabilities and each character is assigned to a set by roulette wheel selection. Otherwise they are treated as integer weights that act as minimum counts, and any remaining characters are distributed round-robin across the sets. Sets with no `prob` share whatever probability is left over, so a policy that only lists names spreads the password evenly across its sets.

Generation and checking treat `require` differently, which is the most common source of a policy that cannot generate a password:

- `Policy.Generate` requires every name in `require` to also appear in `charsets`, matched by exact name. Aliases are not interchangeable here, so pairing `"charsets": ["numbers"]` with `"require": ["digits"]` always fails with `ErrGenerationFailed`, as does requiring `uppercase` while only drawing from `differentiable`.
- `Policy.Check` only tests whether the password contains a character from each required set, so it accepts aliases and sets that are not listed in `charsets`.

### Basic Policy

Used with `Generate("basic")`. It produces a shorter password drawn only from unambiguous characters, which makes it appropriate for passwords that will be transcribed by a human. It is defined as follows:

```json
{
    "length": {
        "min": 9,
        "max": 16
    },
    "charsets": [
        "differentiable"
    ]
}
```

### Default Policy

This is the default policy used by this package, loaded with `Generate("default")`. It is defined as follows:

```json
{
    "length": 16,
    "charsets": [
        {"name": "uppercase", "prob": 0.35},
        {"name": "lowercase", "prob": 0.35},
        {"name": "digits", "prob": 0.2},
        {"name": "symbols", "prob": 0.1}
    ],
    "require": [
        "uppercase",
        "lowercase",
        "digits",
        "symbols"
    ]
}
```

## Character Sets

Character sets are used for password generation and for ensuring specific characters are used in passwords (or are omitted from passwords). The following charactersets are defined by name in this package:

| Name           | Aliases  | Characters                                                | Description                                          |
|----------------|----------|-----------------------------------------------------------|------------------------------------------------------|
| uppercase      | upper    | `ABCDEFGHIJKLMNOPQRSTUVWXYZ`                              | Uppercase characters                                 |
| lowercase      | lower    | `abcdefghijklmnopqrstuvwxyz`                              | Lowercase characters                                 |
| numbers        | digits   | `1234567890`                                              | Digits and numbers                                   |
| symbols        |          | ``!"#$%&'()*+,-./:;<=>?@[\]^_`{\|}~``                     | Punctuation, plus a trailing space character         |
| differentiable |          | `abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRTUVWXY1234567890` | Omits characters that are often confused for numbers |
| alphanumeric   |          | See above                                                 | Combines upper + lower + numbers                     |
| alphasymbolic  |          | See above                                                 | Combines alphanumeric + symbols                      |

The `differentiable` set omits `i`, `l`, `o`, `I`, `O`, `S` and `Z`, which are easily confused with `1`, `0`, `5` and `2`.

You can also specify or override a character set by name using the `define` property in the password policy. Note that `alphanumeric` and `alphasymbolic` are fixed combinations of the built-in sets; redefining `uppercase` does not change what they contain.

## Strength

Strength is a scoring function from "insecure" to "durable" that is used to measure the quality of a password. The strength is computed as follows:

- Length: +1 if ≥ 9 chars, +2 if ≥ 16 chars, +3 if ≥ 32 chars (only the largest applies)
- Charset: +1 for each of uppercase, lowercase, numbers, and symbols that is present
- -1 each time a character is repeated more than twice e.g. AAA or 333
- -1 each time the charset is repeated more than three times in a row e.g. ABCD or 1592

Penalties are applied at every position where they occur, so a long run of one character set is penalized repeatedly. The final score is clamped between 0 and 7.

The strength table by score is as follows:

- 0: Insecure
- 1: Weak
- 2: Soft
- 3: Moderate
- 4: Hard
- 5: Strong
- 6: Robust
- 7: Durable

Passwords are automatically marked "Insecure" if they:

1. Are less than 8 characters
2. Match a word in the common passwords list

The common passwords list is embedded in the package and is matched against the whole password, case-insensitively. It does not detect a common password used as a substring, so `password` scores insecure but `password123` is scored normally.
