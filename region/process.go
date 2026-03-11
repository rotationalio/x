package region

import (
	"os"
	"sync"
)

var (
	processMu     sync.Mutex
	processRegion *Region
)

// Sets the package defined region that determines the region for the process.
func SetProcessRegion(r Region) {
	processMu.Lock()
	processRegion = &r
	processMu.Unlock()
}

// Returns the package defined region for the process. If not set, it attempts to look
// up the region ID from the environment. Returns UNKNOWN if no region is set.
func ProcessRegion() Region {
	processMu.Lock()
	defer processMu.Unlock()
	if processRegion == nil {
		s := os.Getenv("REGION_INFO_ID")
		r, _ := Parse(s)
		processRegion = &r
	}
	return *processRegion
}
