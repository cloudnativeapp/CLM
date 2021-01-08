package prober

import (
	"cloudnativeapp/clm/pkg/probe"
	execprobe "cloudnativeapp/clm/pkg/probe/exec"
	httpprobe "cloudnativeapp/clm/pkg/probe/http"
	tcpprobe "cloudnativeapp/clm/pkg/probe/tcp"
	"cloudnativeapp/clm/pkg/utils"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net"
	"net/http"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
	"time"
)

var probeLog = ctrl.Log.WithName("probe")

type Prober struct {
	exec execprobe.Prober
	http httpprobe.Prober
	tcp  tcpprobe.Prober
}

func NewProber() *Prober {
	const followNonLocalRedirects = false
	return &Prober{
		exec: execprobe.New(),
		http: httpprobe.New(followNonLocalRedirects),
		tcp:  tcpprobe.New(),
	}
}

func (pb *Prober) RunProbeWithRetries(retries int, p *probe.Probe) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < retries; i++ {
		result, output, err = pb.runProbe(p)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

func (pb *Prober) runProbe(p *probe.Probe) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Handler.HTTPGet != nil {
		scheme := strings.ToLower(string(p.Handler.HTTPGet.Scheme))
		host := p.Handler.HTTPGet.Host
		if len(p.Handler.HTTPGet.Namespace) > 0 {
			host = host + "." + p.Handler.HTTPGet.Namespace
		}
		port, err := extractPort(p.Handler.HTTPGet.Port)
		if err != nil {
			return probe.Unknown, "", err
		}
		path := p.Handler.HTTPGet.Path
		probeLog.V(utils.Debug).Info(fmt.Sprintf("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path))
		url := formatURL(scheme, host, port, path)
		headers := buildHeader(p.Handler.HTTPGet.HTTPHeaders)
		probeLog.V(utils.Debug).Info(fmt.Sprintf("HTTP-Probe Headers: %v", headers))
		return pb.http.Probe(url, headers, timeout)

	}
	if p.Handler.TCPSocket != nil {
		port, err := extractPort(p.Handler.TCPSocket.Port)
		if err != nil {
			return probe.Unknown, "", err
		}
		host := p.Handler.TCPSocket.Host
		if len(p.Handler.TCPSocket.Namespace) > 0 {
			host = host + "." + p.Handler.TCPSocket.Namespace
		}
		probeLog.V(utils.Debug).Info(fmt.Sprintf("TCP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout))
		return pb.tcp.Probe(host, port, timeout)
	}
	probeLog.V(utils.Warn).Info("Failed to find probe builder")
	return probe.Unknown, "", fmt.Errorf("missing probe handler")
}

func extractPort(param intstr.IntOrString) (int, error) {
	port := -1
	switch param.Type {
	case intstr.Int:
		port = param.IntValue()
	case intstr.String:
		p, err := strconv.Atoi(param.StrVal)
		if err != nil {
			return -1, err
		}
		port = p
	default:
		return port, fmt.Errorf("intOrString had no kind: %+v", param)
	}
	if port > 0 && port < 65536 {
		return port, nil
	}
	return port, fmt.Errorf("invalid port number: %v", port)
}

func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

func buildHeader(headerList []probe.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}
