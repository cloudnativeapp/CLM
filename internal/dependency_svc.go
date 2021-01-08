package internal

import (
	"cloudnativeapp/clm/pkg/utils"
	"errors"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Strategy when dependency not found in cluster.
	Strategy DependencyStrategy `json:"strategy,omitempty"`
	//  http://example.com/v1/{namespace}/{name}/{version}/content
	Registry Registry `json:"registry,omitempty"`
}

//type Version struct {
//	Min		string `json:"min,omitempty"`
//	Max 	string `json:"max,omitempty"`
//}

type DependencyStatus struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Only when dependency is ready the CRDRelease will continue its installation.
	Phase  DependencyPhase `json:"phase,omitempty"`
	Reason string          `json:"reason,omitempty"`
}

type DependencyStrategy string

const (
	// Pull dependency from registry when it not found in cluster, error will be throw when pull failed.
	PullIfAbsent DependencyStrategy = "PullIfAbsent"
	// Default strategy. CRDRelease will wait until dependency appears.
	WaitIfAbsent DependencyStrategy = "WaitIfAbsent"
	// Throw an error simply.
	ErrorIfAbsent DependencyStrategy = "ErrIfAbsent"
)

type DependencyPhase string

const (
	// Pulling dependency from registry.
	DependencyPulling DependencyPhase = "Pulling"
	// Waiting for dependency to be installed successfully.
	DependencyWaiting DependencyPhase = "Waiting"
	// Error when strategy is ErrorIfAbsent.
	DependencyAbsentErr DependencyPhase = "AbsentError"
	// Pull dependency from registry error.
	DependencyPullingErr DependencyPhase = "PullError"
	// Only when dependency CRDRelease phase is running.
	DependencyRunning DependencyPhase = "Running"
	// Dependency phase abnormal.
	DependencyAbnormal DependencyPhase = "Abnormal"
	// Do not care
	DependencyDontCare DependencyPhase = "DependencyDontCare"
)

var dLog = ctrl.Log.WithName("dependency")

//Install  do install from registry.
func (d Dependency) Install() (interface{}, error) {
	dLog.V(utils.Debug).Info("install dependency", "name", d.Name, "version", d.Version)
	switch d.Strategy {
	case ErrorIfAbsent:
		dLog.V(utils.Warn).Info("dependency strategy is errIfAbsent, throw an error")
		return string(DependencyAbsentErr), errors.New(utils.DependencyAbsentError)
	case PullIfAbsent:
		dLog.V(utils.Debug).Info("dependency strategy is pullIfAbsent")
		if reflect.DeepEqual(d.Registry, Registry{}) {
			dLog.V(utils.Warn).Info("no crd release registry found")
			return string(DependencyPullingErr), errors.New(utils.DependencyPullError)
		}
		return d.Registry.Pull(d.Name, d.Version)
	case WaitIfAbsent:
		fallthrough
	default:
		dLog.V(utils.Warn).Info("dependency strategy is waitIfAbsent")
		return string(DependencyWaiting), errors.New(utils.DependencyWaiting)
	}
}

func (d Dependency) Uninstall() (interface{}, error) {
	panic("can not uninstall dependency now")
}

//Attributes  return the dependency name and version.
func (d Dependency) Attributes() (name, version string) {
	return d.Name, d.Version
}

// DoRecover : Do nothing.
func (d Dependency) DoRecover() (interface{}, error) {
	dLog.V(utils.Debug).Info("can not recover dependency now", "name", d.Name, "version", d.Version)
	return "", nil
}

//DoUpgrade  do upgrade from registry.
func (d Dependency) DoUpgrade() (interface{}, error) {
	dLog.V(utils.Info).Info("dependency upgrade", "release name", d.Name, "target version", d.Version)
	if reflect.DeepEqual(d.Registry, Registry{}) {
		dLog.V(utils.Warn).Info("no crd release registry found")
		return string(DependencyPullingErr), errors.New(utils.DependencyPullError)
	}
	return d.Registry.Pull(d.Name, d.Version)
}

//ConvertStatus : convert crd release phase to dependency status.
func (d Dependency) ConvertStatus(crdReleasePhase string, err error) (interface{}, error) {
	dLog.V(utils.Debug).Info("convert dependency phase to status", "phase", crdReleasePhase)
	status := DependencyStatus{
		Name:    d.Name,
		Version: d.Version,
	}
	if err != nil {
		status.Reason = err.Error()
	}

	switch crdReleasePhase {
	case string(CRDReleaseRunning):
		status.Phase = DependencyRunning
	case "":
		fallthrough
	case string(CRDReleaseInstalling):
		status.Phase = DependencyWaiting
		if len(status.Reason) == 0 {
			status.Reason = utils.DependencyWaiting
		}
	case string(CRDReleaseAbnormal):
		fallthrough
	default:
		status.Phase = DependencyAbnormal
		if len(status.Reason) == 0 {
			status.Reason = utils.DependencyStateAbnormal
		}
	}

	return status, nil
}
