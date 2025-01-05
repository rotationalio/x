package base58_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/base58"
)

func ExampleEncode() {
	data := []byte{193, 65, 211, 109, 255, 213, 186, 58, 6, 122, 175, 146, 99, 34, 19, 124}
	encoded := base58.Encode(data)
	fmt.Println(encoded)
	// Output:
	// Qs8DeAwbRJyRQjFXwm34pT
}

func ExampleDecode() {
	decoded := base58.Decode("Qs8DeAwbRJyRQjFXwm34pT")
	fmt.Println(decoded)
	// Output:
	// [193 65 211 109 255 213 186 58 6 122 175 146 99 34 19 124]
}

func ExampleCheckEncode() {
	data := []byte{193, 65, 211, 109, 255, 213, 186, 58, 6, 122, 175, 146, 99, 34, 19, 124}
	encoded := base58.CheckEncode(data)
	fmt.Println(encoded)
	// Output:
	// 3hADuDuUNzTmuQZJPhELsYw6mqFD
}

func ExampleCheckDecode() {
	decoded, err := base58.CheckDecode("3hADuDuUNzTmuQZJPhELsYw6mqFD")
	fmt.Println(decoded, err)
	// Output:
	// [193 65 211 109 255 213 186 58 6 122 175 146 99 34 19 124] <nil>
}

func TestBase58(t *testing.T) {
	sizes := []int{2, 8, 16, 32, 64, 128, 256, 512, 1024, 4096}
	for _, size := range sizes {
		orig := randBytes(t, size)
		encoded := base58.CheckEncode(orig)
		decoded, err := base58.CheckDecode(encoded)
		assert.Ok(t, err, "could not decode the encoded base58 string")
		assert.True(t, bytes.Equal(orig, decoded), "decoded does not match original bytes")
	}
}

func TestBase58Empty(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		// NOTE: check decode will fail
		assert.Equal(t, "", base58.Encode(nil))
		assert.Equal(t, "3QJmnh", base58.CheckEncode(nil))
		assert.Equal(t, []byte{}, base58.Decode(""))
	})

	t.Run("Array", func(t *testing.T) {
		// NOTE: check decode will fail
		assert.Equal(t, "", base58.Encode([]byte{}))
		assert.Equal(t, "3QJmnh", base58.CheckEncode([]byte{}))
		assert.Equal(t, []byte{}, base58.Decode(""))
	})
}

func TestBase58IncorrectDecode(t *testing.T) {
	// This should not panic but return an empty array
	out := base58.Decode("foo+90?")
	assert.Equal(t, out, []byte{})
}

func TestBase58IncorrectCheckDecode(t *testing.T) {
	testCases := []struct {
		input string
		err   error
	}{
		{
			"3QJmnh", base58.ErrInvalidFormat, // This is the zero-valued array case
		},
		{
			"+9quc2?~", base58.ErrInvalidFormat, // Invalid characters in alphabet
		},
		{
			"", base58.ErrInvalidFormat, // Empty strings have no checksums
		},
		{
			"Qs8DeAwbRJyRQjFXwm34pT", base58.ErrChecksum, // Bad Checksum
		},
	}

	for i, tc := range testCases {
		_, err := base58.CheckDecode(tc.input)
		assert.ErrorIs(t, err, tc.err, "test case %d did not return expected error", i)
	}
}

func randBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	assert.Ok(t, err, "could not generate %d random bytes", n)
	return b
}
