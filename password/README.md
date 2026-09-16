# Password

This package has two primary functionalities:

1. Assessing the strength of a password
2. Generating passwords with a password policy

It also includes a CLI program for quickly generating passwords on the command line.

This package is primarily used by our devops tools to generate passwords for deployments (e.g. superuser passwords, database passwords, etc). It is also used by our authentication tools to force users to create strong passwords with a specific password policy.

## Getting Started with the CLI Program

Install the CLI program with Go:

```
$ go install go.rtnl.ai/x/password/cmd/mkpasswd@latest
```

The `mkpasswd` command should now be available in your `$PATH`. You can set a default policy by creating a policy JSON file in `~/Users/.config/mkpasswd/policies.json`:

```json
{
    "default:" {
        "length": 14,
        "charsets": [
            "differentiable",
            "numbers",
            "symbols",
        ],
        "define": {
            "symbols": "!@#$%^&*()_-+=.><,?"
        },
        "require": [
            "numbers",
            "uppercase",
            "lowercase",
            "symbols"
        ],
        "urlencode": false
    }
}
```

You can also set other named policies in this file to use them if needed. If you need to set a different policies configuration file, set the `$PASSWORD_POLICIES` environment variable with the path to the policy JSON file.

## Policies

Password policies define how passwords are generated and can also be used to verify that a given password meets a specific policy (e.g. to use a different metric other than strength).

This package ships with two policies: `basic` and `strong` and by default the `strong` policy is used if no other policy is specified or defined. These policies can be overridden using the policies configuration file or by specifying an alternate `default` policy.

### Basic Policy

The basic policy is defined as follows:

```json

```

### Strong Policy

This is the default policy used by this package. It is defined as follows:

```json

```

## Character Sets

Character sets are used for password generation and for ensuring specific characters are used in passwords (or are omitted from passwords). The following charactersets are defined by name in this package:

| Name           | Characters                                                 | Description                                          |
|----------------|------------------------------------------------------------|------------------------------------------------------|
| uppercase      | ABCDEFGHIJKLMNOPQRSTUVWXYZ                                 | Uppercase characters                                 |
| lowercase      | abcdefghijklmnopqrstuvwxyz                                 | Lowercase characters                                 |
| numbers        | 1234567890                                                 | Digits and numbers                                   |
| symbols        | !"#$%&'()*+,-./:;<=>?@[\]^_`{\|}~                          | Punctuation includes nonbreaking space               |
| differentiable | abcdefghjkmnpqrstuvwxyz  ABCDEFGHJKLMNPQRTUVWXY 1234567890 | Omits characters that are often confused for numbers |
| alphanumeric   | See above                                                  | Combines upper + lower + numbers (even if redefined) |
| alphasymbolic  | See above                                                  | Combines alphanumeric + symbols (even if redefined)  |

You can also specify or override a character set by name using the `define` property in the password policy.

## Strength

Strength is a scoring function from "weak" to "strong" that is used to measure the quality of a password. The strength is computed as follows:

- Length: +1 if > 8 chars, +2 if > 16 chars, +3 if > 32 chars
- Charset: +1 for each charset if it contains an upper, lower, number, and symbol
- -1 if a character is repeated more than twice e.g. AAA or 333
- -1 if charset is repeated more than three times in a row e.g. ABCD or 1592

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

1. Contain a dictionary word from the common passwords list
2. Are less than 8 characters
