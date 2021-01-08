package resource

import (
	"cloudnativeapp/clm/pkg/cliruntime"
	"cloudnativeapp/clm/pkg/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
)

type Resource struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

//Check  Check the resource.
func (res Resource) Check(log logr.Logger, exist bool) (bool, error) {
	log.V(utils.Debug).Info("try to check resource", "resource", res, "exist", exist)
	allNamespace := false
	if len(res.Namespace) == 0 {
		allNamespace = true
	}
	g := cliruntime.NewGetOption(false, res.Namespace, allNamespace)
	_, err := g.Run([]string{res.Type, res.Name})
	if err != nil {
		log.V(utils.Debug).Info("find resource error", "resource", res, "error", err.Error())
		if errors.IsNotFound(err) && !exist {
			return true, nil
		}
		return false, err
	}
	if exist {
		return true, nil
	}

	return false, nil
}
