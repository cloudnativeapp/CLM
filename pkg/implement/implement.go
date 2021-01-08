package implement

import (
	"cloudnativeapp/clm/pkg/implement/helm"
	"cloudnativeapp/clm/pkg/implement/native"
	"cloudnativeapp/clm/pkg/implement/service"
	"cloudnativeapp/clm/pkg/utils"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Implement struct {
	LocalService *service.Implement `json:"localService,omitempty"`

	Helm *helm.Implement `json:"helm,omitempty"`
	// 本地安装
	Native *native.Implement `json:"native,omitempty"`
}

var iLog = ctrl.Log.WithName("implement")

var serviceFuncMap = make(map[string]func(logr.Logger, service.Implement, map[string]string) (string, error))
var helmFuncMap = make(map[string]func(helm.Implement, map[string]interface{}) (string, error))
var nativeFuncMap = make(map[string]func(logr.Logger, native.Implement, map[string]interface{}) (string, error))

const (
	Install   = "install"
	Uninstall = "uninstall"
	Recover   = "recover"
	Upgrade   = "upgrade"
)

func init() {
	serviceFuncMap[Install] = service.Install
	serviceFuncMap[Uninstall] = service.Uninstall
	serviceFuncMap[Recover] = service.Recover
	serviceFuncMap[Upgrade] = service.Upgrade

	helmFuncMap[Install] = helm.Install
	helmFuncMap[Uninstall] = helm.Uninstall
	helmFuncMap[Recover] = helm.Recover
	helmFuncMap[Upgrade] = helm.Upgrade

	nativeFuncMap[Install] = native.Install
	nativeFuncMap[Uninstall] = native.Uninstall
	nativeFuncMap[Recover] = native.Recover
	nativeFuncMap[Upgrade] = native.Upgrade
}

func (i *Implement) do(action, name, version string, values map[string]interface{}) error {
	iLog.V(utils.Debug).Info("try to do implement", "action", action, "name", name, "values", values)
	if i.LocalService != nil {
		param := service.GetValuesMap(values, name, version)
		if s, err := serviceFuncMap[action](iLog, *i.LocalService,
			param); err != nil {
			iLog.Error(err, fmt.Sprintf("%s implement by service failed", action))
			return err
		} else {
			iLog.V(utils.Info).Info(fmt.Sprintf("service %s implement success", action), "rsp", s)
			return nil
		}
	}
	if i.Helm != nil {
		if s, err := helmFuncMap[action](*i.Helm, values); err != nil {
			iLog.Error(err, fmt.Sprintf("%s implement by helm failed", action))
			return err
		} else {
			iLog.V(utils.Info).Info(fmt.Sprintf("helm %s implement success", action), "rsp", s)
			return nil
		}
	}
	if i.Native != nil {
		if s, err := nativeFuncMap[action](iLog, *i.Native, values); err != nil {
			iLog.Error(err, fmt.Sprintf("%s implement by native failed", action))
			return err
		} else {
			iLog.V(utils.Info).Info(fmt.Sprintf("native %s implement success", action), "rsp", s)
			return nil
		}
	}
	return errors.New(utils.ImplementNotFound)
}

func (i *Implement) Install(name, version string, values map[string]interface{}) error {
	return i.do(Install, name, version, values)
}

func (i *Implement) Uninstall(name, version string, values map[string]interface{}) error {
	return i.do(Uninstall, name, version, values)
}

func (i *Implement) Recover(name, version string, values map[string]interface{}) error {
	return i.do(Recover, name, version, values)
}

func (i *Implement) Upgrade(name, version string, values map[string]interface{}) error {
	return i.do(Upgrade, name, version, values)
}
