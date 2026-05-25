// Package fuzz provides go-fuzz targets and shared wire seeds for purser parsers.
//
// # Fuzz targets (see *_test.go in this directory)
//
//   - FuzzMeta_unmarshal — v1 [models.Meta] canonical round-trip
//   - FuzzSealed_unmarshal — v1 [models.Sealed] canonical round-trip
//   - FuzzParseOpenWire — v1 [models.ParseOpenWire] slice bounds and routing key id
//   - FuzzParseKeyID_agreesWithUnmarshal — [models.ParseKeyIDFromSealed] vs full unmarshal
//   - FuzzParseKeyID — [locker.Locker.ParseKeyID] per registered edition
//   - FuzzRegistry_ParseKeyID — [registry.ParseKeyID] edition dispatch
//
// # Running
//
// From the repo root: ./purser/fuzz/fuzzpurser.sh (see fuzzpurser.sh for smoke and single-target runs).
// Manual runs use -fuzz=TargetName$ and -timeout above -fuzztime.
//
// Commit new inputs under testdata/fuzz/FuzzTargetName/ when a run finds a crasher.
//
// # Edition seeds
//
// editionSeeds in this file is the single registration point for locker-edition fuzz
// corpora. Good wire is always produced via [registry.FromSeed] and [locker.Locker.Seal].
// [AddEditionWireSeeds] registers every edition; [AddV1WireSeeds] registers v1 only.
//
// Adding a registry edition (e.g. v2):
//
//  1. Add a row to editionSeeds (edition, namespace, negatives).
//  2. Ensure TestEditionSeedsCoverRegistry passes.
//  3. Add AddV2WireSeeds (or rely on AddEditionWireSeeds) and v2 parser fuzz as needed.
//  4. Re-run fuzz targets and commit any new testdata entries.
package fuzz

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/keyring/registry"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/wire"
)

//=============================================================================
// Constants
//=============================================================================

// PreambleBytes is the fixed sealed-row header before variable-length meta.
const PreambleBytes = wire.PreambleBytes

// fuzzPlaintext is the plaintext sealed into edition good-wire seeds.
const fuzzPlaintext = "plain"

// fixedFuzzSeed is deterministic locker seed bytes for reproducible fuzz corpora.
var fixedFuzzSeed = func() []byte {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i)
	}
	return seed
}()

//=============================================================================
// Edition seed registration
//=============================================================================

// editionSeed holds fuzz corpus metadata for one [registry] locker edition.
type editionSeed struct {
	edition   string
	namespace string
	negatives func(good []byte) [][]byte
}

// editionSeeds is the central registration for edition-targeting fuzz corpora.
// Add a row when shipping a new registry edition.
var editionSeeds = map[string]editionSeed{
	constv1.Edition: {edition: constv1.Edition, namespace: "ns", negatives: v1WireNegatives},
}

//=============================================================================
// Locker and wire seeds (unversioned dispatchers)
//=============================================================================

// NewLocker returns a locker for edition using the fuzz-fixed seed.
func NewLocker(tb testing.TB, edition string) locker.Locker {
	tb.Helper()

	if _, ok := editionSeeds[edition]; !ok {
		tb.Fatalf("fuzz: no locker seed registered for edition %q", edition)
	}

	lck, err := registry.FromSeed(edition, fixedFuzzSeed)
	assert.Ok(tb, err)
	return lck
}

// SealGoodWire returns sealed wire for edition via [registry.FromSeed] and [locker.Locker.Seal].
func SealGoodWire(tb testing.TB, edition string) []byte {
	tb.Helper()

	row, ok := editionSeeds[edition]
	if !ok {
		tb.Fatalf("fuzz: no wire seed registered for edition %q", edition)
	}

	lck := NewLocker(tb, edition)
	out, err := lck.Seal(row.namespace, []byte(fuzzPlaintext))
	assert.Ok(tb, err)
	return out
}

// AddWireSeeds registers one good wire blob and negative seeds on f.
func AddWireSeeds(f *testing.F, good []byte, negatives [][]byte) {
	f.Add(good)
	for _, n := range negatives {
		f.Add(append([]byte(nil), n...))
	}
}

// AddEditionWireSeeds registers good wire and negatives for every [editionSeeds] row.
func AddEditionWireSeeds(f *testing.F) {
	for edition := range editionSeeds {
		addEditionWireSeeds(f, edition)
	}
}

// addEditionWireSeeds registers good wire and negatives for one edition.
func addEditionWireSeeds(f *testing.F, edition string) {
	row, ok := editionSeeds[edition]
	if !ok {
		f.Fatalf("fuzz: no wire seed registered for edition %q", edition)
	}
	good := SealGoodWire(f, edition)
	AddWireSeeds(f, good, row.negatives(good))
}

//=============================================================================
// v1 wire seeds
//=============================================================================

// GoodV1Wire returns valid v1 sealed wire ([SealGoodWire] for [constv1.Edition]).
func GoodV1Wire(tb testing.TB) []byte {
	tb.Helper()
	return SealGoodWire(tb, constv1.Edition)
}

// AddV1WireSeeds registers good v1 wire and v1 negative corpora on f.
func AddV1WireSeeds(f *testing.F) {
	addEditionWireSeeds(f, constv1.Edition)
}

// AddV1OpenWireSeeds registers [models.ParseOpenWire] seeds (wire + namespace).
func AddV1OpenWireSeeds(f *testing.F) {
	good := SealGoodWire(f, constv1.Edition)
	f.Add(good, "ns")
	f.Add(good, "other-ns")
	f.Add([]byte{}, "")
	f.Add(good[:PreambleBytes], "ns")
	f.Add(append([]byte(nil), good[:len(good)-1]...), "ns")
}

// v1WireNegatives returns shared negative PURS v1 wire seeds derived from good wire.
func v1WireNegatives(good []byte) [][]byte {
	badVer := []byte("PURS")
	badVer = append(badVer, 99, 0, 0)

	return [][]byte{
		{},
		[]byte("PURS"),
		good[:PreambleBytes],
		append([]byte(nil), good[:len(good)-1]...),
		[]byte("PURS\x01\x00#\x01\x00 00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		badVer,
	}
}
