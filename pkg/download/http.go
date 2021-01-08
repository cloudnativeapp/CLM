package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func HttpGet(p string) (string, error) {
	u, err := url.Parse(p)
	if err != nil || u.Scheme == "" {
		p, _ = filepath.Abs(p)
		if _, err := os.Stat(p); err != nil {
			return "", err
		}
	} else {
		return HttpAndFileSchemeDownload(p, u)
	}

	return "", nil
}

func HttpAndFileSchemeDownload(p string, url *url.URL) (string, error) {
	tr := &http.Transport{}
	tr.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	c := &http.Client{Transport: tr}
	c.Timeout = 3600 * time.Second
	resp, err := c.Get(p)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		_, f := filepath.Split(url.Path)

		path := filepath.Join(os.TempDir(), f)
		out, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", err
		}
		return path, nil

	} else {
		return "", errors.New(fmt.Sprintf("download failed, http response:%s.", resp.Status))
	}
}
