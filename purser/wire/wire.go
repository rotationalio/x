// Package wire defines the shared purser sealed-row preamble (magic + format version).
package wire

const (
	// Magic is the four-byte preamble for all purser sealed rows.
	Magic = "PURS"

	// MagicLen is the byte length of [Magic].
	MagicLen = 4

	// VersionOffset is the index of the format-version byte immediately after [Magic].
	VersionOffset = MagicLen

	// PreambleBytes is magic(4) + formatVersion(1) + metaLen u16 BE(2).
	PreambleBytes = MagicLen + 1 + 2
)
