package recover

import (
	"cloudnativeapp/clm/pkg/cliruntime"
	"cloudnativeapp/clm/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type Recover struct {
	Job string `json:"job,omitempty"`
}

func (r Recover) DoRecover(log logr.Logger) error {
	if len(r.Job) != 0 {
		n := cliruntime.NewApplyOptions(nil, r.Job, false)
		err := n.Run()
		if err != nil {
			log.Error(err, "do recover by job failed")
			return err
		}
		log.V(utils.Info).Info("run recover job succeed")
		return nil
	}
	return errors.New("no recover method found")
}
