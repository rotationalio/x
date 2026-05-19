package suite_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// TestSuite_wireRoundTrip checks the one-byte wire encoding round-trips for every
// non-Unknown entry in suite.Names. Adding a new suite to the Names table extends
// coverage automatically, so this test must stay table-driven.
func TestSuite_wireRoundTrip(t *testing.T) {
	for i, name := range suite.Names {
		id := suite.ID(i)
		if id == suite.Unknown {
			continue
		}
		t.Run(name, func(t *testing.T) {
			raw, err := id.MarshalBinary()
			assert.Ok(t, err)
			assert.Equal(t, []byte{byte(id)}, raw)

			var got suite.ID
			assert.Ok(t, got.UnmarshalBinary(raw))
			assert.Equal(t, id, got)
			assert.True(t, got.Valid(), "%s must report Valid", name)
			assert.Equal(t, name, got.String())
		})
	}
}

// TestSuite_UnmarshalBinary_rejectsBadInput covers nil receiver and non-single-byte wire (what Meta parsing relies on).
func TestSuite_UnmarshalBinary_rejectsBadInput(t *testing.T) {
	var p *suite.ID
	assert.ErrorIs(t, p.UnmarshalBinary([]byte{1}), perrors.ErrNilSuiteID)

	var id suite.ID
	assert.ErrorIs(t, id.UnmarshalBinary(nil), perrors.ErrInvalidSuiteWire)
	assert.ErrorIs(t, id.UnmarshalBinary([]byte{1, 2}), perrors.ErrInvalidSuiteWire)
}

// TestSuite_Parse exercises the coercion paths that actually show up at API boundaries (config / wire decode helpers).
func TestSuite_Parse(t *testing.T) {
	cases := []struct {
		name    string
		in      any
		want    suite.ID
		wantErr error
	}{
		{
			name: "wire_name",
			in:   "x25519_hkdf_sha256_aes256_gcm",
			want: suite.X25519HKDFSHA256AES256GCM,
		},
		{
			name: "string_decimal_same_byte_as_suite",
			in:   "1",
			want: suite.X25519HKDFSHA256AES256GCM,
		},
		{
			name: "uint8",
			in:   uint8(1),
			want: suite.X25519HKDFSHA256AES256GCM,
		},
		{
			name: "int",
			in:   1,
			want: suite.X25519HKDFSHA256AES256GCM,
		},
		{
			name: "int64",
			in:   int64(1),
			want: suite.X25519HKDFSHA256AES256GCM,
		},
		{
			name:    "unknown_name",
			in:      "totally_unknown_suite",
			wantErr: perrors.ErrUnknownSuiteName,
		},
		{
			name:    "decimal_string_out_of_uint8",
			in:      "256",
			wantErr: perrors.ErrUnknownSuiteName,
		},
		{
			name:    "int_out_of_range",
			in:      256,
			wantErr: perrors.ErrInvalidSuiteValue,
		},
		{
			name:    "int64_out_of_range",
			in:      int64(256),
			wantErr: perrors.ErrInvalidSuiteValue,
		},
		{
			name:    "negative_int",
			in:      -1,
			wantErr: perrors.ErrInvalidSuiteValue,
		},
		{
			name:    "wrong_go_type",
			in:      struct{}{},
			wantErr: perrors.ErrInvalidSuiteInput,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := suite.Parse(tc.in)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.Ok(t, err)
			assert.Equal(t, tc.want, got)
			assert.True(t, got.Valid())
		})
	}
}
