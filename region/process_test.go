package region_test

import (
	"os"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/region"
)

func TestProcessRegion(t *testing.T) {
	t.Run("Environment", func(t *testing.T) {
		region.ResetProcessRegion()

		// Restore environment after the test
		curEnv, exists := os.LookupEnv("REGION_INFO_ID")
		defer func() {
			if exists {
				os.Setenv("REGION_INFO_ID", curEnv)
			} else {
				os.Unsetenv("REGION_INFO_ID")
			}
		}()

		// Set environment variable
		os.Setenv("REGION_INFO_ID", "2840291")

		// Test ProcessRegion
		process := region.ProcessRegion()
		assert.Equal(t, region.GCP_US_EAST_1B, process)

		// Change environment variable
		os.Setenv("REGION_INFO_ID", "2756191")

		// Test ProcessRegion
		process = region.ProcessRegion()
		assert.Equal(t, region.GCP_US_EAST_1B, process)

	})

	t.Run("SetProcessRegion", func(t *testing.T) {
		defer region.ResetProcessRegion()

		// Test SetProcessRegion
		region.SetProcessRegion(region.GCP_US_EAST_1B)
		assert.Equal(t, region.GCP_US_EAST_1B, region.ProcessRegion())
	})

	t.Run("Unknown", func(t *testing.T) {
		defer region.ResetProcessRegion()

		// Test Unknown
		assert.Equal(t, region.UNKNOWN, region.ProcessRegion(), "wanted %s, got %s", region.UNKNOWN, region.ProcessRegion())
	})

	t.Run("Invalid", func(t *testing.T) {
		defer region.ResetProcessRegion()

		// Restore environment after the test
		curEnv, exists := os.LookupEnv("REGION_INFO_ID")
		defer func() {
			if exists {
				os.Setenv("REGION_INFO_ID", curEnv)
			} else {
				os.Unsetenv("REGION_INFO_ID")
			}
		}()

		// Set environment variable
		os.Setenv("REGION_INFO_ID", "foo")

		// Test Invalid
		assert.Equal(t, region.UNKNOWN, region.ProcessRegion(), "wanted %s, got %s", region.UNKNOWN, region.ProcessRegion())
	})

}
