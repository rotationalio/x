package lockertest

// Negative conformance tests for locker.Locker (see [LockerConforms]).

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/locker"
)

//=============================================================================
// Tests: negative Locker conformance
//=============================================================================

// TestLockerConformance_negative runs table-style subtests; each pairs a broken locker
// with a conformance check that should detect the defect.
func TestLockerConformance_negative(t *testing.T) {
	t.Run("key_id_empty", func(t *testing.T) {
		err := checkKeyIDNonEmpty(func() (locker.Locker, error) {
			return &emptyKeyIDLocker{}, nil
		})
		assert.Error(t, err, "expected conformance check to fail for empty KeyID")
	})

	t.Run("key_id_not_defensive_copy", func(t *testing.T) {
		lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("shared-kid"))
		assert.Ok(t, err)
		shared := append([]byte(nil), lck.KeyID()...)
		err = checkKeyIDDefensiveCopy(func() (locker.Locker, error) {
			return &sharedKeyIDLocker{Locker: lck, kid: shared}, nil
		})
		assert.Error(t, err, "expected conformance check to fail for shared KeyID slice")
	})

	t.Run("open_wrong_namespace", func(t *testing.T) {
		lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("open-ns"))
		assert.Ok(t, err)
		err = checkOpenWrongNamespace(func() (locker.Locker, error) {
			return &permissiveOpenLocker{Locker: lck}, nil
		})
		assert.Error(t, err, "expected conformance check to fail for permissive Open")
	})

	t.Run("parse_key_id_accepts_empty", func(t *testing.T) {
		lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("parse-empty"))
		assert.Ok(t, err)
		err = checkParseKeyIDRejectsEmpty(func() (locker.Locker, error) {
			return &permissiveParseKeyIDLocker{Locker: lck}, nil
		})
		assert.Error(t, err, "expected conformance check to fail for permissive ParseKeyID")
	})
}

//=============================================================================
// Broken locker fakes
//=============================================================================

type emptyKeyIDLocker struct{}

func (emptyKeyIDLocker) Version() uint8  { return 0 }
func (emptyKeyIDLocker) Edition() string { return "broken" }
func (emptyKeyIDLocker) Recipe() string  { return "" }
func (emptyKeyIDLocker) Context() string { return "" }
func (emptyKeyIDLocker) KeyID() []byte   { return nil }
func (emptyKeyIDLocker) Seal(string, []byte) ([]byte, error) {
	return nil, nil
}
func (emptyKeyIDLocker) Open(string, []byte) ([]byte, error) { return nil, nil }
func (emptyKeyIDLocker) ParseKeyID([]byte) ([]byte, error)   { return nil, nil }

type sharedKeyIDLocker struct {
	locker.Locker
	kid []byte
}

func (s *sharedKeyIDLocker) KeyID() []byte {
	return s.kid
}

type permissiveOpenLocker struct {
	locker.Locker
}

func (p *permissiveOpenLocker) Open(namespace string, wire []byte) ([]byte, error) {
	_ = namespace
	return p.Locker.Open("ns-a", wire)
}

type permissiveParseKeyIDLocker struct {
	locker.Locker
}

func (p *permissiveParseKeyIDLocker) ParseKeyID(wire []byte) ([]byte, error) {
	if len(wire) == 0 {
		return []byte("accepted"), nil
	}
	return p.Locker.ParseKeyID(wire)
}
