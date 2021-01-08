package helm

import (
	"cloudnativeapp/clm/pkg/download"
	"cloudnativeapp/clm/pkg/helmsdk"
	"github.com/pkg/errors"
	"time"
)

type Implement struct {
	//Repository string `json:"repository,omitempty"`
	Wait    bool `json:"wait,omitempty"`
	Timeout int  `json:"timeout,omitempty"`
}

func Install(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Upgrade(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Uninstall(i Implement, values map[string]interface{}) (string, error) {
	releaseName := values["releaseName"].(string)
	chartPath := values["chartPath"].(string)
	namespace := values["namespace"].(string)
	if len(releaseName) == 0 {
		return "", errors.New("empty releaseName")
	}
	if len(namespace) == 0 {
		namespace = "default"
	}
	return helmsdk.Uninstall(chartPath, releaseName, namespace)
}

func Recover(i Implement, values map[string]interface{}) (string, error) {
	return doInstallOrUpgrade(i, values)
}

func Status(i Implement, values map[string]interface{}) (string, error) {
	releaseName := values["releaseName"].(string)
	namespace := values["namespace"].(string)
	s, err := helmsdk.Status(releaseName, namespace)
	if err != nil {
		return "", err
	}
	return string(s.Info.Status), nil
}

func doInstallOrUpgrade(i Implement, values map[string]interface{}) (string, error) {
	chartPath := values["chartPath"].(string)
	releaseName := values["releaseName"].(string)
	namespace := values["namespace"].(string)
	var vals map[string]interface{}
	if values["chartValues"] != nil {
		v, err := decodeValues(values["chartValues"])
		if err != nil {
			return "", err
		}
		vals = v
	}
	if len(chartPath) == 0 || len(releaseName) == 0 {
		return "", errors.New("empty chartPath or releaseName")
	}
	if len(namespace) == 0 {
		namespace = "default"
	}

	chartPath, err := download.HttpGet(chartPath)
	if err != nil {
		return "", err
	}

	installed, err := helmsdk.Exist(chartPath, namespace)
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
		return helmsdk.Upgrade(chartPath, releaseName, namespace, vals, i.Wait, timeoutSecond)
	} else {
		return helmsdk.Install(chartPath, releaseName, namespace, vals, i.Wait, timeoutSecond)
	}
}

func decodeValues(values interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	result = values.(map[string]interface{})
	return result, nil
}
