package supervisor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/zeebo/errs"

	"storj.io/common/version"
)

var errProcessAlreadyStarted = errors.New("process already started")

type Process struct {
	cmd *exec.Cmd

	binPath string
	args    []string
}

func NewProcess(binPath string, args []string) *Process {
	return &Process{binPath: binPath, args: args}
}

// start starts the process.
// It returns errProcessAlreadyStarted if the process is already started.
func (p *Process) start(ctx context.Context) (err error) {
	if p.cmd != nil {
		return errProcessAlreadyStarted
	}

	p.cmd = exec.CommandContext(ctx, p.binPath, p.args[1:]...)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	return p.cmd.Start()
}

// wait waits for the process to finish.
func (p *Process) wait() error {
	return p.cmd.Wait()
}

// exit stops the process by sending an interrupt signal.
func (p *Process) exit() error {
	return p.cmd.Process.Signal(os.Interrupt)
}

// Version returns the version of the process.
func (p *Process) Version(ctx context.Context) (version.SemVer, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out, err := exec.CommandContext(ctx, p.binPath, "version").CombinedOutput()
	if err != nil {
		return version.SemVer{}, err
	}

	return parseVersion(out)
}

func parseVersion(out []byte) (version.SemVer, error) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		prefix := "Version: "
		if strings.HasPrefix(line, prefix) {
			line = line[len(prefix):]
			return version.NewSemVer(line)
		}
	}
	return version.SemVer{}, errs.New("unable to determine binary version")
}