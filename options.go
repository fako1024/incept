package incept

import "time"

// WithShutdownGraceTime sets a custom grace time for the shutdown procedure
func WithShutdownGraceTime(shutdownGraceTime time.Duration) func(*Incept) {
	return func(i *Incept) {
		i.shutdownGraceTime = shutdownGraceTime
	}
}

// WithExitFn sets a custom function to execute once all children and the master
// process is done
func WithExitFn(exitFn func(code int)) func(*Incept) {
	return func(i *Incept) {
		i.exitFn = exitFn
	}
}
