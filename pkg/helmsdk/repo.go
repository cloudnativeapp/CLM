package helmsdk

import (
	"cloudnativeapp/clm/pkg/utils"
	"context"
	"github.com/go-logr/logr"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

type repoAddOptions struct {
	name                 string
	url                  string
	username             string
	password             string
	forceUpdate          bool
	allowDeprecatedRepos bool

	certFile              string
	keyFile               string
	caFile                string
	insecureSkipTLSverify bool

	repoFile  string
	repoCache string

	// Deprecated, but cannot be removed until Helm 4
	deprecatedNoUpdate bool
}

func Add(name, url, username, password string, log logr.Logger) error {
	config := cli.New()
	o := &repoAddOptions{
		name:      name,
		url:       url,
		username:  username,
		password:  password,
		repoFile:  config.RepositoryConfig,
		repoCache: config.RepositoryCache,
	}
	return o.run(log)
}

func (o *repoAddOptions) run(log logr.Logger) error {
	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(o.repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(o.repoFile, filepath.Ext(o.repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(o.repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	c := repo.Entry{
		Name:                  o.name,
		URL:                   o.url,
		Username:              o.username,
		Password:              o.password,
		CertFile:              o.certFile,
		KeyFile:               o.keyFile,
		CAFile:                o.caFile,
		InsecureSkipTLSverify: o.insecureSkipTLSverify,
	}

	if f.Has(o.name) {
		existing := f.Get(o.name)
		if c != *existing {

			// The input coming in for the name is different from what is already
			// configured. Return an error.
			return errors.Errorf("repository name (%s) already exists, please specify a different name", o.name)
		}

		// The add is idempotent so do nothing
		log.V(utils.Info).Info(o.name + "already exists with the same configuration, skipping")
		return nil
	}

	r, err := repo.NewChartRepository(&c, getter.All(cli.New()))
	if err != nil {
		return err
	}

	if o.repoCache != "" {
		r.CachePath = o.repoCache
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", o.url)
	}

	f.Update(&c)

	if err := f.WriteFile(o.repoFile, 0644); err != nil {
		return err
	}
	log.V(utils.Info).Info(o.name + "has been added to repositories")
	return nil
}
