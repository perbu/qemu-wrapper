package tuntap

import (
	"fmt"
	"os/exec"
)

type Executor interface {
	Run(path string, args ...string) ([]byte, error)
}

type executor struct {
}

func newExecutor() *executor {
	return &executor{}
}

func (e *executor) Run(path string, args ...string) ([]byte, error) {
	cmd := exec.Command(path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing %s: %w (output: %s)", cmd.Path, err, out)
	}
	return out, nil
}
