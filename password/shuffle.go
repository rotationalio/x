package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Shuffles the characters in the string using the Fisher-Yates algorithm with
// cryptographically secure random number generation.
func Shuffle(s string) string {
	runes := []rune(s)
	n := len(runes)

	for i := n - 1; i > 0; i-- {
		// Generate a secure random index j such that 0 <= j <= i.
		max := big.NewInt(int64(i + 1))
		jb, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(fmt.Errorf("failed to generate secure random index: %w", err))
		}

		j := int(jb.Int64())
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}
