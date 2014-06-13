package main

type Health struct {
	prevHealthy bool
	Healthy     bool

	previous int
	current  int

	threshold int
}

func NewHealth(threshold int) *Health {
	return &Health{false, false, 0, 0, threshold}
}

// Watch takes two channels of booleans. The "in" channel represents passed
// health checks (true for passed, false for failed)
func (h *Health) Watch(in chan bool) chan bool {
	out := make(chan bool, 1)

	go func() {
		for status := range in {
			h.previous = h.current

			// add or subtract bad checks
			if status && h.current > 0 {
				h.current--
			} else if !status && h.current <= h.threshold {
				h.current++
			}

			// detect if we're currently past either threshold
			h.prevHealthy = h.Healthy

			if h.current == h.threshold {
				h.Healthy = false
			} else if h.current == 0 {
				h.Healthy = true
			}

			// detect state changes
			if h.prevHealthy != h.Healthy {
				out <- h.Healthy
			}
		}
	}()

	return out
}
