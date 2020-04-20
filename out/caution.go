package out

import "sync"

// Counts the number of caution messages until a threshold is reached
type counter struct {
	sync.Mutex
	counts    map[string]uint
	threshold uint
}

// initialize the counter object
func (c *counter) init() {
	c.counts = make(map[string]uint)
}

// keep track of the number of messages logged, if seen for the first time, return true,
// otherwise if greater than the threshold, remove it so the next time the message is
// observed it returns true.
func (c *counter) log(msg string) bool {
	c.Lock()
	defer c.Unlock()

	c.counts[msg]++
	if c.counts[msg] == 1 {
		return true
	}

	if c.counts[msg] > c.threshold-1 {
		delete(c.counts, msg)
	}
	return false
}
