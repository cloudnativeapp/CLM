package plugin

import (
	"errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Iplugin interface {
	// Get the name, version of the plugin.
	Attributes() (string, string)
	// Install the plugin, return status and error.
	Install() (interface{}, error)
	// Uninstall the plugin, return status and error.
	Uninstall() (interface{}, error)
	// Recover the plugin, return status and error.
	DoRecover() (interface{}, error)
	// Upgrade the plugin, return status and error.
	DoUpgrade() (interface{}, error)
	// Get the status of plugin, return status and error.
	//GetStatus() (interface{}, error)
	// convert the phase and error to status of actual plugin, throw error when phase abnormal
	ConvertStatus(string, error) (interface{}, error)
}

// StatusSet : Set status to plugin.
// Inputs: name, version, status.
// Return: ok, error
type StatusSet func(string, string, interface{}) (bool, error)

// StatusGet : Get the status of the plugin.
// Inputs: name, version.
// Return: action needed, plugin phase got, error.
type StatusGet func(string, string) (Action, string, error)

type Action string

const (
	// Need to install the plugin.
	NeedInstall Action = "NeedInstall"
	// Need to recover plugin.
	NeedRecover Action = "NeedRecover"
	// Need to uninstall the plugin.
	NeedUninstall Action = "NeedUninstall"
	//
	NeedNothing Action = "NeedNothing"
	// Need to upgrade the plugin.
	NeedUpgrade Action = "NeedUpgrade"
	// Need to convert phase to plugin status.
	NeedConvert Action = "NeedConvert"
)

var pluginLog = ctrl.Log.WithName("plugin")

//CheckPlugins : Check the status of all plugins.
//Return true when all status of plugins are ready.
func CheckPlugins(ps []Iplugin, setStatus StatusSet, getStatus StatusGet) (bool, error) {
	ready := true
	var errs []error
	for _, p := range ps {
		name, version := p.Attributes()
		ok, e := CheckPlugin(p, setStatus, getStatus)
		if e != nil {
			pluginLog.Error(e, "check plugin error", "name", name, "version", version)
			errs = append(errs, e)
			ready = false
		} else if !ok {
			ready = false
		}
	}

	return ready, NormalizedErrors(errs)
}

//NormalizedErrors All errors normalized as one.
func NormalizedErrors(errs []error) error {
	var msg string
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	for _, e := range errs {
		msg = msg + ";" + e.Error()
	}

	return errors.New(msg)
}

//CheckPlugin Check whether the plugin is ready.
func CheckPlugin(c Iplugin, setStatus StatusSet, getStatus StatusGet) (ready bool, err error) {
	var status interface{}
	n, v := c.Attributes()
	act, phase, err := getStatus(n, v)
	switch act {
	case NeedInstall:
		status, err = c.Install()
	case NeedRecover:
		pluginLog.Info("try to restore plugin", "name", n, "version", v)
		status, err = c.DoRecover()
	case NeedUninstall:
		pluginLog.Info("try to uninstall plugin", "name", n, "version", v)
		status, err = c.Uninstall()
	case NeedUpgrade:
		pluginLog.Info("try to upgrade plugin", "name", n, "version", v)
		status, err = c.DoUpgrade()
	case NeedConvert:
		fallthrough
	default:
		pluginLog.Info("try to convert status directly", "name", n, "version", v)
		status, _ = c.ConvertStatus(phase, err)
	}

	ok, err := setStatus(n, v, status)
	if err != nil {
		pluginLog.Error(err, "set plugin status error", "name", n, "version", v)
	}
	return ok, err
}
