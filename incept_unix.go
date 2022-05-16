package incept

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ignoreSIGCHLD() {
	signal.Ignore(syscall.SIGCHLD)
}

func shutdown(grace time.Duration) error {
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		return err
	}
	time.Sleep(grace)
	return syscall.Kill(os.Getpid(), syscall.SIGKILL)
}

func fork() error {
	argv0, wd, err := getBinaryPaths()
	if nil != err {
		return err
	}

	env := append(os.Environ(), fmt.Sprintf("%s=TRUE", envChildMarker))
	if _, err = os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   env,
		Files: getFDs(),
		Sys:   &syscall.SysProcAttr{},
	}); err != nil {
		return err
	}

	return nil
}

func getFDs() []*os.File {
	return []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
	}
}
