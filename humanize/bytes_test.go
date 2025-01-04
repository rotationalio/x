package humanize_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/humanize"
)

func TestByteUnits(t *testing.T) {
	// Test that bytes are correctly converted to the expected units.
	tests := []struct {
		bytes uint64
		units string
		value float64
	}{
		{bytes: 0, units: "B", value: 0},
		{bytes: 1, units: "B", value: 1},
		{bytes: 1023, units: "B", value: 1023},
		{bytes: 1024, units: "KiB", value: 1},
		{bytes: (1024 * 1024) + (1024 * 512), units: "MiB", value: 1.5},
		{bytes: 1024 * 1024 * 1024, units: "GiB", value: 1},
		{bytes: 1024 * 1024 * 1024 * 1024, units: "TiB", value: 1},
	}

	for _, test := range tests {
		units, value := humanize.FromBytes(test.bytes)
		assert.Equal(t, test.units, units, "wrong units")
		assert.Equal(t, test.value, value, "wrong value")
	}
}
