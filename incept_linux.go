//go:build freebsd || linux
// +build freebsd linux

package incept

import "syscall"

var sysProcAttr = syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
