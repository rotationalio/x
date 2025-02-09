# Semantic Versioning

Utilities for identifying, parsing, and comparing semantic version strings as
specified by [Semantic Versioning 2.0.0](https://semver.org/).

To test if a string is a valid semantic version:

```go
semver.Valid("1.3.1")
// true

semver.Valid("1.foo")
// false
```

To parse a string into its semantic version components:

```go
vers, err := semver.Parse("1.3.1-alpha")
```

To compare the precedence of two versions:

```go
semver.Compare(a, b)
// or
a.Compare(b)
```

[Precedence](https://semver.org/#spec-item-11) defines how versions are compared to each other when ordered (e.g. which is the later version number). The output of compare is:

- `-1`: a < b or b has the higher precedence
- `0`: a == b or a and b have the same precedence (but not necessarily equal strings)
- `1`: a > b or a has the higher precedence

A `Range` is a a specification of semantic version boundaries, and are used to check if a semantic version is part of a range or not. For example:

```go
rng := semver.Range(">=1.3.1")
rng.Contains("1.4.8")
// true
rng.Contains("1.2.9")
// false
```

Ranges are denoted by an operator:

| Operator | Definition | Example | Notes |
|---|---|---|---|
| = | Match exact version | =1.2.3 | No operator is interpreted as = |
| > | Match higher precedence versions | >1.2.3 |  |
| < | Match lower precedence versions | <1.2.3 |  |
| >= | Match exact or higher precedence versions | >=1.2.3 |  |
| <=  | Match exact or lower precedence versions | <=1.2.3 |  |
| ~ | Match "reasonably close to" | ~1.2.3 | is >=1.2.3 && <1.3.0 |
| ^ | Match "compatible with" | ^1.2.3 | is >=1.2.3 && <2.0.0 |
| x or X | Any version | 1.x.x | is >=1.0.0 && <2.0.0 |
| * | Any version | 1.2.* | is >=1.2.0 && <1.3.0 |
| - | Hyphenated range | 1.2.3 - 2.3.4 | is >=1.2.3 && <=2.3.4 |

Ranges can be combined with `&&` (AND) and `||` (OR) operators for more complex range definitions.