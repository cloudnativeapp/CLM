package native

import (
	"cloudnativeapp/clm/pkg/cliruntime"
	"github.com/go-logr/logr"
	"strings"
)

type Implement struct {
	IgnoreError bool `json:"ignoreError,omitempty"`
}

func Install(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	return applyAction(log, i, values)
}

func Uninstall(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	return deleteAction(log, i, values)
}

func Upgrade(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	return applyAction(log, i, values)
}

func Recover(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	return applyAction(log, i, values)
}

func Status(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	return "", nil
}

func applyAction(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	urls, yamls := getUrlAndStream(values)
	n := cliruntime.NewApplyOptions(urls, yamls, i.IgnoreError)
	err := n.Run()
	if err != nil {
		log.Error(err, "native apply error")
		return "", err
	}
	return "native apply success", nil
}

func deleteAction(log logr.Logger, i Implement, values map[string]interface{}) (string, error) {
	urls, yamls := getUrlAndStream(values)
	d, err := cliruntime.NewDeleteOptions(urls, yamls)
	if err != nil {
		log.Error(err, "new native delete command failed")
		return "", err
	}

	err = d.RunDelete()
	if err != nil {
		log.Error(err, err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return err.Error(), nil
		}
		if i.IgnoreError {
			return err.Error(), nil
		}
		return "", err
	}

	return "native delete success", nil
}

func getUrlAndStream(values map[string]interface{}) ([]string, string) {
	var urls []string
	var yamlStr string
	if v, ok := values["urls"]; ok {
		if v != nil {
			for _, i := range v.([]interface{}) {
				urls = append(urls, i.(string))
			}
		}
	}
	if v, ok := values["yaml"]; ok {
		yamlStr = v.(string)
	}

	return urls, yamlStr
}
