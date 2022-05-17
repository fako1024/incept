package incept

import "time"

// WithShutdownGraceTime sets a custom grace time for the shutdown procedure
func WithShutdownGraceTime(shutdownGraceTime time.Duration) func(*Incept) {
	return func(i *Incept) {
		i.shutdownGraceTime = shutdownGraceTime
	}
}
