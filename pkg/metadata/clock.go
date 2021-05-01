package metadata

import "time"

// Clock give time
type Clock struct {
	now time.Time
}

// Now return current time
func (c *Clock) Now() time.Time {
	if c == nil {
		return time.Now()
	}
	return c.now
}
