# Base58

Base58 is a binary-to-text encoding scheme that uses an alphabet of 58 characters to represent data and is commonly used in Bitcoin and with Travel Addresses. The alphabet chosen purposefully avoids similar looking letters, therefore it is useful for things that have to be read out loud by humans. However it is more inefficient than base64 because it has fewer characters and parsing is more awkward because the base is not a power of 2.

This library was ported from [github.com/trisacrypto/trisa/pkg/openvasp/traddr](https://pkg.go.dev/github.com/trisacrypto/trisa/pkg/openvasp/traddr).

Use the `Encode` and `Decode` functions for simple base58 handling:

```go
data := []byte{193, 65, 211, 109, 255, 213, 186, 58, 6, 122, 175, 146, 99, 34, 19, 124}
encoded := base58.Encode(data)
decoded := base58.Decode(encoded)

bytes.Equal(data, decoded)
// true
```

Note that `Decode` will not report any errors, if it encounters a problem it will just return an empty byte array. Because of that, it is recommended that you use `CheckEncode` and `CheckDecode` which also adds a version number and a checksum to the decoding process:

```go
data := []byte{193, 65, 211, 109, 255, 213, 186, 58, 6, 122, 175, 146, 99, 34, 19, 124}
encoded := base58.CheckEncode(data)
decoded, err := base58.CheckDecode(encoded)

err == nil
// true

bytes.Equal(data, decoded)
// true
```

This will result in a slightly longer encoding, but the checksum will ensure the data has not been corrupted when decoding.
