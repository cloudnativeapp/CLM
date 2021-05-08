package helmsdk

import (
	"cloudnativeapp/clm/pkg/download"
	"cloudnativeapp/clm/pkg/utils"
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

var hLog = ctrl.Log.WithName("helm-sdk")

func Install(chartPath, releaseName, namespace string, vals map[string]interface{}, wait bool, timeouts time.Duration) (string, error) {
	hLog.V(utils.Debug).Info("try to install", "chartPath", chartPath, "releaseName", releaseName,
		"namespace", namespace, "values", vals, "wait", wait, "timeouts", timeouts)
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return "", err
	}
	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	if wait {
		client.Wait = true
		client.Timeout = timeouts
	}
	charts, err := loader.Load(chartPath)
	if err != nil {
		return "", err
	}
	hLog.V(utils.Debug).Info("charts load", "charts", charts)
	results, err := client.Run(charts, vals)
	if err != nil {
		return "", err
	}
	hLog.V(utils.Debug).Info("charts installed", "result", results)
	return results.Name, nil
}

func Uninstall(chartPath, releaseName, namespace string) (string, error) {
	hLog.V(utils.Debug).Info("try to uninstall", "chartPath", chartPath,
		"releaseName", releaseName, "namespace", namespace)
	chartPathLocal, err := download.HttpGet(chartPath)
	if err != nil {
		return "", err
	}
	if len(chartPathLocal) == 0 {
		chartPathLocal, err = LocateChart(chartPath, namespace)
		if err != nil {
			return "", err
		}
	}
	if exist, err := Exist(chartPathLocal, namespace); err != nil {
		return "", err
	} else if !exist {
		hLog.V(utils.Info).Info("no chart found", "chartPath", chartPath,
			"releaseName", releaseName, "namespace", namespace)
		return "", nil
	}
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return "", err
	}
	c := action.NewUninstall(actionConfig)
	r, err := c.Run(releaseName)
	if err != nil {
		return "", err
	}
	hLog.V(utils.Debug).Info("charts uninstalled", "result", r)
	return r.Release.Name, nil
}

func Status(releaseName, namespace string) (*release.Release, error) {
	hLog.V(utils.Debug).Info("try to status", "releaseName", releaseName, "namespace", namespace)
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewStatus(actionConfig)
	r, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func List(namespace string) ([]*release.Release, error) {
	hLog.V(utils.Debug).Info("try to list", "namespace", namespace)
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewList(actionConfig)
	r, err := client.Run()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func Upgrade(chartPath, releaseName, namespace string, vals map[string]interface{}, wait bool, timeouts time.Duration) (string, error) {
	hLog.V(utils.Debug).Info("try to upgrade", "chartPath", chartPath, "releaseName", releaseName,
		"namespace", namespace, "values", vals, "wait", wait, "timeouts", timeouts)
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return "", err
	}
	charts, err := loader.Load(chartPath)
	if err != nil {
		return "", err
	}
	hLog.V(utils.Debug).Info("charts load", "charts", charts)
	client := action.NewUpgrade(actionConfig)
	client.ReuseValues = true
	if wait {
		client.Wait = true
		client.Timeout = timeouts
	}
	r, err := client.Run(releaseName, charts, vals)
	if err != nil {
		return "", err
	}
	hLog.V(utils.Debug).Info("charts upgrade", "result", r)
	return r.Name, nil
}

func Exist(chartPath, namespace string) (bool, error) {
	target, err := loader.Load(chartPath)
	if err != nil {
		return false, err
	}
	hLog.V(utils.Debug).Info("charts load", "charts", target)
	charts, err := List(namespace)
	if err != nil {
		return false, err
	}
	for _, chart := range charts {
		if chart.Chart.Metadata.Name == target.Metadata.Name &&
			chart.Chart.Metadata.Version == target.Metadata.Version {
			return true, nil
		}
	}

	return false, nil
}

func getSdkLog() func(format string, v ...interface{}) {
	return func(format string, v ...interface{}) {
		hLog.V(utils.Debug).Info(fmt.Sprintf(format, v...))
	}
}

func getActionConfig(namespace string) (*action.Configuration, error) {
	settings := cli.New()
	settings.EnvVars()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), getSdkLog()); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

// Locate charts and download
func LocateChart(chart, namespace string) (string, error) {
	ch := action.ChartPathOptions{}
	os.Setenv("HELM_NAMESPACE", namespace)
	cp, err := ch.LocateChart(chart, cli.New())
	if err != nil {
		return "", err
	}
	return cp, nil
}
