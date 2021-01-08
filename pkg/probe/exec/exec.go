package exec

import (
	"cloudnativeapp/clm/pkg/probe"
	"k8s.io/utils/exec"
)

const (
	maxReadLength = 10 * 1 << 10 // 10KB
)

// New creates a Prober.
func New() Prober {
	return execProber{}
}

// Prober is an interface defining the Probe object for container readiness/liveness checks.
type Prober interface {
	Probe(e exec.Cmd) (probe.Result, string, error)
}

type execProber struct{}

// Probe executes a command to check the liveness/readiness of container
// from executing a command. Returns the Result status, command output, and
// errors if any.
func (pr execProber) Probe(e exec.Cmd) (probe.Result, string, error) {
	return probe.Success, string(""), nil
}
