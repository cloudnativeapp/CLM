package condition

import (
	"cloudnativeapp/clm/pkg/check/resource"
	"cloudnativeapp/clm/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type Condition struct {
	// All resources should not exist.
	ResourceNotExist []resource.Resource `json:"resourceNotExist,omitempty"`
	// All resources should exist.
	ResourceExist []resource.Resource `json:"resourceExist,omitempty"`
	// Strategy when condition does not met
	Strategy Strategy `json:"strategy,omitempty"`
}

type Strategy string

const (
	// When condition check failed, the module can not be managed by clm.
	External Strategy = "PullIfAbsent"
	// When condition check failed, we take it as a ready module.
	Import Strategy = "Import"
)

//Check  do condition check
func (c Condition) Check(log logr.Logger) (bool, error) {
	log.V(utils.Debug).Info("try to check condition", "condition", c)
	if len(c.ResourceNotExist) != 0 {
		for _, r := range c.ResourceNotExist {
			if ok, err := r.Check(log, false); err != nil {
				log.Error(err, "condition check error")
				return false, err
			} else if !ok {
				log.Error(errors.New("resource exists"), "condition check failed",
					"name", r.Name, "type", r.Type)
				return false, nil
			}
		}
	}

	if len(c.ResourceExist) != 0 {
		for _, r := range c.ResourceExist {
			if ok, err := r.Check(log, true); err != nil {
				log.Error(err, "condition check error")
				return false, err
			} else if !ok {
				log.Error(errors.New("resource does not exist"), "condition check failed",
					"name", r.Name, "type", r.Type)
				return false, nil
			}
		}
	}

	return true, nil
}
