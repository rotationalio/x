/*
Package memzero provides byte-slice zeroing for sensitive buffers.
*/
package memzero

// Zero overwrites b with zeros.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
