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

// log a message, returns true if the threshold has been reached and the message should
// be printed (and resets the counter).
//
// TODO: Reverse so that the first message is printed and subsequent messages are ignored.
func (c *counter) log(msg string) bool {
	c.Lock()
	defer c.Unlock()

	c.counts[msg]++
	if c.counts[msg] > c.threshold {
		delete(c.counts, msg)
		return true
	}

	return false
}
