/*
This library helps generate random strings for a variety of use cases, e.g. to generate
passwords, API keys, token strings, one time codes, etc. It uses the `crypto/rand`
package to securely generate the random strings and can use a variety of alphabets for
random generation. The generation is handled in the most efficient way possible to
ensure that the library does not cause unnecessary bottlenecks or memory usage.
*/
package randstr

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	mrand "math/rand"
	"strings"
)

const (
	uppercase  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase  = "abcdefghijklmnopqrstuvwxyz"
	alphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphanum   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	passwords  = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{};:\|,.<>/?;`
	consonants = "bcdfghjklmnpqrstvwxz"
	vowels     = "aeiou"
	idxbits    = 6
	idxmask    = 1<<idxbits - 1
	idxmax     = 63 / idxbits
)

// Alpha generates a random string of n characters that only includes upper and
// lowercase letters (no symbols or digits).
func Alpha(n int) string {
	return Generate(n, alphabet)
}

func AlphaUpper(n int) string {
	return Generate(n, uppercase)
}

func AlphaLower(n int) string {
	return Generate(n, lowercase)
}

// AlphaNumeric generates a random string of n characters that includes upper and
// lowercase letters and the digits 0-9.
func AlphaNumeric(n int) string {
	return Generate(n, alphanum)
}

// Password generates a random string of n characters that includes upper and lowercase
// letters, the digits 0-9, and the special characters !@#$%^&*()_+-=[]{};:\|,.<>/?;
func Password(n int) string {
	return Generate(n, passwords)
}

// Word generates a random string of n special characters with only consonants and
// vowels to mimic a fake word. Words are not guaranteed to be unique.
func Word(n int) string {
	if n < 1 {
		return ""
	}

	chars := make([]rune, 0, n)
	numConsonants := (n / 2) + 1

	chars = append(chars, []rune(Generate(numConsonants, consonants))...)
	chars = append(chars, []rune(Generate(n-numConsonants, vowels))...)

	// Shuffle the chars to make a word
	word := strings.Builder{}
	word.Grow(n)
	for remain := len(chars); remain > 0; remain-- {
		idx := mrand.Intn(len(chars))
		word.WriteRune(chars[idx])
		chars = append(chars[:idx], chars[idx+1:]...)
	}

	return word.String()
}

// Generate a random string of n characters from the character set defined by chars. It
// uses as efficient a method of generation as possible, using a string builder to
// prevent multiple allocations and a 6 bit mask to select 10 random letters at a time
// to add to the string. This method would be far faster if it used math/rand src and
// the Int63() function, but for API key generation it is important to use a
// cryptographically random generator.
//
// See: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func Generate(n int, chars string) string {
	sb := strings.Builder{}
	sb.Grow(n)

	for i, cache, remain := n-1, CryptoRandInt(), idxmax; i >= 0; {
		if remain == 0 {
			cache, remain = CryptoRandInt(), idxmax
		}

		if idx := int(cache & idxmask); idx < len(chars) {
			sb.WriteByte(chars[idx])
			i--
		}

		cache >>= idxbits
		remain--
	}

	return sb.String()
}

func CryptoRandInt() uint64 {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("cannot generate random number: %w", err))
	}
	return binary.BigEndian.Uint64(buf)
}
