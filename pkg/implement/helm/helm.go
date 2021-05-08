package helm

import (
	"cloudnativeapp/clm/pkg/download"
	"cloudnativeapp/clm/pkg/helmsdk"
	"github.com/pkg/errors"
	"time"
)

type Implement struct {
	Wait         bool   `json:"wait,omitempty"`
	Timeout      int    `json:"timeout,omitempty"`
	IgnoreError  bool   `json:"ignoreError,omitempty"`
	Repositories []Repo `json:"repositories,omitempty"`
}

type Repo struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	UserName string `json:"username,omitempty"`
	PassWord string `json:"password,omitempty"`
}

func Install(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Upgrade(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Uninstall(i Implement, values map[string]interface{}) (string, error) {
	releaseName, ok := values["releaseName"].(string)
	if !ok && len(releaseName) == 0 {
		return "", errors.New("release name needed")
	}
	chartPath, ok := values["chartPath"].(string)
	if !ok {
		return "", errors.New("chart path needed")
	}
	namespace, ok := values["namespace"].(string)
	if !ok && len(namespace) == 0 {
		namespace = "default"
	}
	result, err := helmsdk.Uninstall(chartPath, releaseName, namespace)
	if err != nil && !i.IgnoreError {
		return result, err
	}
	return result, nil
}

func Recover(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Status(i Implement, values map[string]interface{}) (string, error) {
	releaseName, ok := values["releaseName"].(string)
	if !ok && len(releaseName) == 0 {
		return "", errors.New("release name needed")
	}
	namespace, ok := values["namespace"].(string)
	if !ok && len(namespace) == 0 {
		namespace = "default"
	}
	s, err := helmsdk.Status(releaseName, namespace)
	if err != nil {
		return "", err
	}
	return string(s.Info.Status), nil
}

func doInstallOrUpgrade(i Implement, values map[string]interface{}) (string, error) {
	releaseName, ok := values["releaseName"].(string)
	if !ok && len(releaseName) == 0 {
		return "", errors.New("release name needed")
	}
	chartPath, ok := values["chartPath"].(string)
	if !ok {
		return "", errors.New("chart path needed")
	}
	namespace, ok := values["namespace"].(string)
	if !ok && len(namespace) == 0 {
		namespace = "default"
	}
	var vals map[string]interface{}
	if values["chartValues"] != nil {
		v, err := decodeValues(values["chartValues"])
		if err != nil {
			return "", err
		}
		vals = v
	}

	chartPathLocal, err := download.HttpGet(chartPath)
	if err != nil {
		return "", err
	}
	if len(chartPathLocal) == 0 {
		chartPathLocal, err = helmsdk.LocateChart(chartPath, namespace)
		if err != nil {
			return "", err
		}
	}

	installed, err := helmsdk.Exist(chartPathLocal, namespace)
	if err != nil {
		return "", err
	}
	var timeoutSecond time.Duration
	if i.Timeout <= 0 {
		timeoutSecond = 60 * time.Second
	} else {
		timeoutSecond = time.Duration(i.Timeout) * time.Second
	}
	if installed {
		return helmsdk.Upgrade(chartPathLocal, releaseName, namespace, vals, i.Wait, timeoutSecond)
	} else {
		return helmsdk.Install(chartPathLocal, releaseName, namespace, vals, i.Wait, timeoutSecond)
	}
}

func decodeValues(values interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	result, ok := values.(map[string]interface{})
	if !ok {
		return nil, errors.New("chartValues format error")
	}
	return result, nil
}
