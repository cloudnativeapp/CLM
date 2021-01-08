package precheck

import "cloudnativeapp/clm/pkg/check/resource"

type Precheck struct {
	// All resources should not exist.
	ResourceNotExist []resource.Resource `json:"resourceNotExist,omitempty"`
	// All resources should exist.
	ResourceExist []resource.Resource `json:"resourceExist,omitempty"`
}
