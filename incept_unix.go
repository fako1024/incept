package incept

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func shutdownPID(pid int, grace time.Duration) error {

	// TODO: Actually implement a proper grace time / KILL mechanism
	// defer func() {
	// 	time.Sleep(grace)
	// 	syscall.Kill(pid, syscall.SIGKILL)
	// }()

	return syscall.Kill(pid, syscall.SIGTERM)
}

func fork() (*os.Process, error) {
	argv0, wd, err := getBinaryPaths()
	if nil != err {
		return nil, err
	}

	env := append(os.Environ(), fmt.Sprintf("%s=TRUE", envChildMarker))
	p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   env,
		Files: getFDs(),
		Sys:   &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM},
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

func getFDs() []*os.File {
	return []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
	}
}
