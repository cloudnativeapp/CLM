package internal

import (
	"bytes"
	"cloudnativeapp/clm/pkg/cliruntime"
	"cloudnativeapp/clm/pkg/implement/service"
	"cloudnativeapp/clm/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

type Registry struct {
	// Http default
	Protocol RegistryProtocol `json:"protocol,omitempty"`
	// Host, ip or hostname.
	Host string `json:"host"`
	// The http path, version/namespace/releaseName/releaseVersion default.
	Path string `json:"relativePath,omitempty"`
	// Registry version.
	Version   string `json:"version,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// Parameters to render the crd release.
	Params map[string]string `json:"renderParams,omitempty"`
}

type RegistryProtocol string

const (
	Http  RegistryProtocol = "http"
	Https RegistryProtocol = "https"
)

//Pull  pull crd release from registry and apply with cli-runtime.
func (r Registry) Pull(name, version string) (interface{}, error) {
	dLog.V(utils.Debug).Info("start pull crd release from registry", "name", name, "version", version)
	ip, port, err := decodeHost(r.Host)
	if err != nil {
		dLog.Error(err, "decode host failed", "host", r.Host)
		return string(DependencyPullingErr), err
	}
	var path string
	if len(r.Path) > 0 {
		path, err = render("path", r.Path, r.Params)
		if err != nil {
			dLog.Error(err, "render path failed", "path", r.Path, "params", r.Params)
			return string(DependencyPullingErr), err
		}

		path = fmt.Sprintf("http://%s:%s/%s", ip, port, path)
	} else {
		path = fmt.Sprintf("http://%s:%s/%s/%s/%s/%s", ip, port,
			r.Version, r.Namespace, name, version)
	}

	if resp, err := do(path, r.Params); err != nil {
		dLog.Error(err, fmt.Sprintf("pull error, ip:%s", ip))
		return string(DependencyPullingErr), err
	} else {
		crdrelease, err := render(name, resp, r.Params)
		if err != nil {
			dLog.Error(err, "render crd release failed")
			return string(DependencyPullingErr), err
		}
		dLog.V(utils.Debug).Info("pull crd release", "content", crdrelease)
		y, err := yaml.JSONToYAML([]byte(crdrelease))
		if err != nil {
			dLog.Error(err, "convert json to yaml failed")
			return string(DependencyPullingErr), err
		}
		// 使用create接口
		n := cliruntime.NewApplyOptions(nil, string(y), false)
		err = n.Run()
		if err != nil {
			dLog.Error(err, "native apply error")
			return string(DependencyPullingErr), err
		}
		return string(DependencyPulling), nil
	}
}

func render(name, input string, params map[string]string) (string, error) {
	dLog.V(utils.Debug).Info("start render object", "name", name, "input", input, "params", params)
	if params == nil || len(params) == 0 {
		dLog.V(utils.Debug).Info("do not need to render")
		return input, nil
	}
	tmpl, err := template.New(name).Parse(input)
	if err != nil {
		dLog.Error(err, "template parse failed")
		return "", err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, params)
	if err != nil {
		dLog.Error(err, "template exec failed")
		return "", err
	}
	return buf.String(), nil
}

func do(path string, params map[string]string) (string, error) {
	dLog.V(utils.Debug).Info("do pull crd release", "path", path, "params", params)
	if resp, err := service.Do("get", path, nil, nil); err != nil {
		dLog.Error(err, "http get error")
		return "", err
	} else {
		defer resp.Body.Close()
		if content, err := ioutil.ReadAll(resp.Body); err != nil {
			dLog.Error(err, "http get read content error")
			return "", err
		} else {
			if result, err := disableEscape(content); err != nil {
				dLog.Error(err, "disable escape error")
				return "", err
			} else {
				return result, nil
			}
		}
	}
}

func disableEscape(input []byte) (string, error) {
	data := make(map[string]interface{})
	if err := json.Unmarshal(input, &data); err != nil {
		return "", err
	}
	newStr, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(newStr), err
}

func decodeHost(host string) (ip, port string, err error) {
	str := strings.Split(host, ":")
	ip = str[0]
	if len(str) > 2 {
		err := errors.New("decode host failed")
		return ip, port, err
	}
	if len(str) == 2 {
		port = str[1]
	} else {
		port = "80"
	}
	return ip, port, err
}
