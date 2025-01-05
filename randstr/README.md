# Random Strings

This library helps generate random strings for a variety of use cases, e.g. to generate passwords, API keys, token strings, one time codes, etc. It uses the `crypto/rand` package to securely generate the random strings and can use a variety of alphabets for random generation. The generation is handled in the most efficient way possible to ensure that the library does not cause unnecessary bottlenecks or memory usage.

You can generate a string with any arbitrary alphabet as follows:

```go
// generates 16 characters with just vowels
randstr.Generate(16, "aeiou")
```

Or use the helpers:

```go
// Generate 16 char string with only upper and lowercase letters
randstr.Alpha(16)

// Generate 16 char string with upper and lowercase letters + digits
randstr.AlphaNumeric(16)

// Generate 16 char string with alpha numeric + special characters
randstr.Password(16)

// Generate 6 char string that interleaves vowels with consonants
randstr.Word(6)
```