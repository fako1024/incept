package incept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	defaultShutdownGraceTime = 15 * time.Second
	envChildMarker           = "INCEPT_CHILD"
	binaryBackupFilename     = ".replace.tmp"
)

// Incept denotes an instance of a process reloader
type Incept struct {
	shutdownGraceTime time.Duration

	pid              int
	argv0            string
	workingDir       string
	binaryBackupPath string

	exitFn func(code int)
}

// New instanciates a nnew Incept instance (should be called as soon as possible)
// When called from the parent / main process, it will simply fork a subprocess and wait forever
// When called in the child process, it will continue and serve as the replacable process for
// any updates in the future
func New(options ...func(*Incept)) (*Incept, error) {

	// Determine binary path and working directory
	argv0, wd, err := getBinaryPaths()
	if nil != err {
		return nil, err
	}

	// Initialize a new incept instance with default parameters
	i := &Incept{
		shutdownGraceTime: defaultShutdownGraceTime,
		argv0:             argv0,
		workingDir:        wd,
		binaryBackupPath:  filepath.Join(wd, filepath.Dir(argv0), binaryBackupFilename),

		exitFn: func(code int) {
			os.Exit(code)
		},
	}

	// Execute functional options, if any
	for _, opt := range options {
		opt(i)
	}

	// Fork a child process if this is the parent and wait forever, otherwise continue
	if !i.IsChild() {

		signalChild := make(chan os.Signal, 1)
		defer close(signalChild)
		signal.Notify(signalChild, syscall.SIGUSR2, syscall.SIGCHLD)
		defer signal.Stop(signalChild)

		p, err := fork()
		if err != nil {
			return nil, err
		}
		i.pid = p.Pid

		for {

			// Process incoming signal
			// TODO: Make OS specific and handle in extra method
			s := (<-signalChild)
			switch s {

			// If SIGCHLD was received, the child terminated (or was terminated). Propagate
			// child return value and exit
			case syscall.SIGCHLD:
				var ws syscall.WaitStatus
				if _, err := syscall.Wait4(i.pid, &ws, syscall.WNOHANG, nil); err != nil {
					return nil, err
				}

				i.exitFn(ws.ExitStatus())
				return i, err

			// If SIGUSR2 was received, the child indicates that it wants to be restarted
			// Fork a new child and terminate the old one
			case syscall.SIGUSR2:
				p, err = fork()
				if err != nil {
					return nil, err
				}
				if err := shutdownPID(i.pid, i.shutdownGraceTime); err != nil {
					return nil, err
				}
				<-signalChild
				var ws syscall.WaitStatus
				if _, err := syscall.Wait4(i.pid, &ws, 0, nil); err != nil {
					return nil, err
				}

				// Remove the old binary
				if err := os.RemoveAll(i.binaryBackupPath); err != nil {
					return nil, err
				}
				i.pid = p.Pid
			}
		}
	}

	return i, nil
}

// IsChild returns if this is a child process
func (i *Incept) IsChild() bool {
	return os.Getenv(envChildMarker) != ""
}

// Update performs the update, provided a new binary to load and an optional list
// of functions to execute prior to the replacement (e.g. server web server shutdown)
func (i *Incept) Update(binary []byte, shutdownFn ...func() error) error {

	// Perform a stat() call to extract the file permissions of the current
	// binary for transfer to the new one
	stat, err := os.Stat(i.argv0)
	if err != nil {
		return err
	}

	// Rename the currently running binary to a temporary file
	if err := os.Rename(i.argv0, i.binaryBackupPath); err != nil {
		return err
	}

	// Write the new binary
	if err := ioutil.WriteFile(i.argv0, binary, stat.Mode().Perm()); err != nil {
		return err
	}

	// Ensure the update is performed after returning from this method
	// TODO: Either handle errors properly somehow or implement a zero-downtime way of replacing
	// the binary in case there is a webserver (otherwise the in-line execution here would Kill
	// any existing connection)
	defer func() {
		// TODO: This is probably still racy and only works because the return -> potential server handler
		// is much faster than the execution of the shutdownFns. Maybe there's better ways
		go i.update(i.binaryBackupPath, shutdownFn...)
	}()

	return nil
}

/////////////////////////////////////////////////////////////

func (i *Incept) update(binaryBackupPath string, shutdownFn ...func() error) error {

	// Execute shutdown handlers, if any
	for _, fn := range shutdownFn {
		if err := fn(); err != nil {
			return err
		}
	}

	// Indicate to the parent / master process that the child is ready to be replaced
	return syscall.Kill(os.Getppid(), syscall.SIGUSR2)
}

func verifyChecksum(data []byte, expectedChecksum []byte) error {
	hash := sha256.New()
	if n, err := hash.Write(data); err != nil || len(data) != n {
		return fmt.Errorf("invalid data submitted for hashing")
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	if checksum != string(expectedChecksum) {
		return fmt.Errorf("mismatching checksums: expected `%s`, got `%s`", expectedChecksum, checksum)
	}

	return nil
}

func getBinaryPaths() (argv0 string, wd string, err error) {
	argv0, err = exec.LookPath(os.Args[0])
	if nil != err {
		return
	}
	if _, err = os.Stat(argv0); nil != err {
		return
	}
	wd, err = os.Getwd()

	return
}
