package purser

// Shared helpers used across purser and locker implementations.

// Zero overwrites b with zeros. Use to wipe key material after use.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
