package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

type Implement struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Install   Spec   `json:"install"`
	Uninstall Spec   `json:"uninstall"`
	Upgrade   Spec   `json:"upgrade"`
	Recover   Spec   `json:"recover"`
	Status    Spec   `json:"status"`
}

type Spec struct {
	// Support Http
	Protocol     string     `json:"protocol,omitempty"`
	RelativePath string     `json:"relativePath,omitempty"`
	Values       url.Values `json:"values,omitempty"`
	// Supports post(default) put get delete patch
	Method string `json:"method,omitempty"`
}

type ImplementRsp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

func GetValuesMap(input map[string]interface{}, name, version string) map[string]string {
	result := make(map[string]string)
	result["name"] = name
	result["version"] = version
	if input != nil {
		for k, v := range input {
			result[k] = v.(string)
		}
	}
	return result
}

func Install(log logr.Logger, svc Implement, params map[string]string) (string, error) {
	rsp, err := doAction(log, svc.Name, svc.Namespace, params, svc.Install)
	if err != nil {
		return "", err
	}
	if rsp.Code != 200 {
		err = errors.New("install failed:" + rsp.Msg)
		log.Error(err, " message: "+rsp.Msg)
		return "", err
	}
	return rsp.Msg, nil
}

func Uninstall(log logr.Logger, svc Implement, params map[string]string) (string, error) {
	rsp, err := doAction(log, svc.Name, svc.Namespace, params, svc.Uninstall)
	if err != nil {
		return "", err
	}
	if rsp.Code != 200 {
		err = errors.New("uninstall failed:" + rsp.Msg)
		log.Error(err, " message: "+rsp.Msg)
		return "", err
	}
	return rsp.Msg, nil
}

func Upgrade(log logr.Logger, svc Implement, params map[string]string) (string, error) {
	rsp, err := doAction(log, svc.Name, svc.Namespace, params, svc.Upgrade)
	if err != nil {
		return "", err
	}
	if rsp.Code != 200 {
		err = errors.New("upgrade failed:" + rsp.Msg)
		log.Error(err, " message: "+rsp.Msg)
		return "", err
	}
	return rsp.Msg, nil
}

func Recover(log logr.Logger, svc Implement, params map[string]string) (string, error) {
	rsp, err := doAction(log, svc.Name, svc.Namespace, params, svc.Recover)
	if err != nil {
		return "", err
	}
	if rsp.Code != 200 {
		err = errors.New("recover failed:" + rsp.Msg)
		log.Error(err, " message: "+rsp.Msg)
		return "", err
	}
	return rsp.Msg, nil
}

func Status(log logr.Logger, svc Implement, params map[string]string) (string, error) {
	rsp, err := doAction(log, svc.Name, svc.Namespace, params, svc.Status)
	if err != nil {
		return "", err
	}
	if rsp.Code != 200 {
		err = errors.New("status failed:" + rsp.Msg)
		log.Error(err, " message: "+rsp.Msg)
		return "", err
	}
	return rsp.Msg, nil
}

func getClientSet() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(ctrl.GetConfigOrDie())
}

func getService(name, namespace string) (ip string, port int32, err error) {
	k8sClient, err := getClientSet()
	if err != nil {
		return "", 0, err
	}
	service, err := k8sClient.CoreV1().Services(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return "", 0, err
	}

	// 目前支持tcp和http
	for _, port := range service.Spec.Ports {
		if port.Protocol == "TCP" || port.Protocol == "HTTP" {
			return service.Spec.ClusterIP, port.Port, nil
		}
	}

	return "", 0, errors.New("no tcp or http port found")
}

func doAction(log logr.Logger, name, namespace string, params map[string]string, spec Spec) (rsp ImplementRsp, err error) {
	ip, port, err := getService(name, namespace)
	if err != nil {
		log.Error(err, "install implement by service failed")
		return rsp, err
	}

	if resp, err := Do(spec.Method, fmt.Sprintf("http://%s:%d/%s", ip, port, spec.RelativePath), spec.Values,
		params); err != nil {
		log.Error(err, fmt.Sprintf("http %s error, ip:%s", spec.Method, ip))
		return rsp, err
	} else {
		defer resp.Body.Close()
		var rsp ImplementRsp
		if content, err := ioutil.ReadAll(resp.Body); err != nil {
			log.Error(err, fmt.Sprintf("http %s read content error, ip:%s", spec.Method, ip))
			return rsp, err
		} else if err := json.Unmarshal(content, &rsp); err != nil {
			log.Error(err, fmt.Sprintf("http %s unmarshal content error, ip:%s", spec.Method, ip))
			return rsp, err
		} else {
			log.Info(fmt.Sprintf("http %s success, ip:%s, rsp:%s", spec.Method, ip, string(content)))
			return rsp, nil
		}
	}
}

func Do(method, url string, values url.Values, params map[string]string) (resp *http.Response, err error) {
	req, err := http.NewRequest(getHttpMethod(method), url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return http.DefaultClient.Do(req)
}

func getHttpMethod(method string) string {
	switch strings.ToLower(method) {
	case strings.ToLower(http.MethodGet):
		return http.MethodGet
	case strings.ToLower(http.MethodDelete):
		return http.MethodDelete
	case strings.ToLower(http.MethodPut):
		return http.MethodPut
	default:
		return http.MethodPost
	}
}
