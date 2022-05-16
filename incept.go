package incept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

	argv0      string
	workingDir string
}

// New instanciates a nnew Incept instance (should be called as soon as possible)
// When called from the parent / main process, it will simply fork a subprocess and wait forever
// When called in the child process, it will continue and serve as the replacable process for
// any updates in the future
func New() (*Incept, error) {

	// Determine binary path and working directory
	argv0, wd, err := getBinaryPaths()
	if nil != err {
		return nil, err
	}

	// Fork a child process if this is the parent and wait forever, otherwise continue
	if os.Getenv(envChildMarker) == "" {
		if err := fork(); err != nil {
			return nil, err
		}

		// Ensure that forked children do not end up zombie after being terminated
		ignoreSIGCHLD()

		// Wait forever
		select {}
	}

	return &Incept{
		shutdownGraceTime: defaultShutdownGraceTime,
		argv0:             argv0,
		workingDir:        wd,
	}, nil
}

// ShutdownGraceTime sets a custom grace time for the shutdown procedure
func (i *Incept) ShutdownGraceTime(shutdownGraceTime time.Duration) *Incept {
	i.shutdownGraceTime = shutdownGraceTime
	return i
}

// Update performs the update, provided a new binary to load and an optional list
// of functions to execute prior to the replacement (e.g. server web server shutdown)
func (i *Incept) Update(binary []byte, shutdownFn ...func() error) error {

	stat, err := os.Stat(i.argv0)
	if err != nil {
		return err
	}

	// Rename the currently running binary to a temporary file
	binaryBackupPath := filepath.Join(i.workingDir, filepath.Dir(i.argv0), binaryBackupFilename)
	if err := os.Rename(i.argv0, binaryBackupPath); err != nil {
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
		go i.update(binaryBackupPath, shutdownFn...)
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

	// Fork a new process based on the (now replaced) binary
	if _, err := os.StartProcess(i.argv0, os.Args, &os.ProcAttr{
		Dir:   i.workingDir,
		Env:   os.Environ(),
		Files: getFDs(),
		Sys:   &syscall.SysProcAttr{},
	}); err != nil {
		return err
	}

	// Remove the old binary
	if err := os.Remove(binaryBackupPath); err != nil {
		return err
	}

	// (Self-)Terminate the running child / process
	return shutdown(i.shutdownGraceTime)
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
