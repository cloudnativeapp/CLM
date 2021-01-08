package tcp

import (
	"cloudnativeapp/clm/pkg/probe"
	"cloudnativeapp/clm/pkg/utils"
	"net"
	"strconv"
	"time"
)

// New creates Prober.
func New() Prober {
	return tcpProber{}
}

// Prober is an interface that defines the Probe function for doing TCP readiness/liveness checks.
type Prober interface {
	Probe(host string, port int, timeout time.Duration) (probe.Result, string, error)
}

type tcpProber struct{}

// Probe returns a ProbeRunner capable of running an TCP check.
func (pr tcpProber) Probe(host string, port int, timeout time.Duration) (probe.Result, string, error) {
	return DoTCPProbe(net.JoinHostPort(host, strconv.Itoa(port)), timeout)
}

// DoTCPProbe checks that a TCP socket to the address can be opened.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
// This is exported because some other packages may want to do direct TCP probes.
func DoTCPProbe(addr string, timeout time.Duration) (probe.Result, string, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		// Convert errors to failures to handle timeouts.
		return probe.Failure, err.Error(), nil
	}
	err = conn.Close()
	if err != nil {
		probe.PLog.V(utils.Warn).Info("Unexpected error closing TCP probe socket: %v (%#v)", err, err)
	}
	return probe.Success, "ready", nil
}
