package precheck

import (
	"context"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CRDCheck struct {
	Conflict []CRD `json:"conflict,omitempty"`
	Required []CRD `json:"required,omitempty"`
}

type CRD struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

//Check 为true才能继续安装
func (c *CRDCheck) Check() (bool, error) {
	scheme := runtime.NewScheme()
	k8sclient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		log.Printf("new k8s client err %v", err)
		return false, err
	}

	o := &client.ListOptions{}
	var list v1.CustomResourceDefinitionList
	v1.AddToScheme(scheme)
	if err = k8sclient.List(context.Background(), &list, o); err != nil {
		log.Printf("list crd err %v", err)
		var list v1beta1.CustomResourceDefinitionList
		v1beta1.AddToScheme(scheme)
		if err = k8sclient.List(context.Background(), &list, o); err != nil {
			log.Printf("list crd err %v", err)
			return false, err
		}
		_, e := doCheckV1beta1(list.Items, c.Conflict, c.Required)
		return e, nil
	}
	_, e := doCheckV1(list.Items, c.Conflict, c.Required)
	return e, nil
}

func doCheckV1beta1(crds []v1beta1.CustomResourceDefinition, conflicts, exists []CRD) (bool, bool) {
	conflict := true
	exist := true
	for _, c := range conflicts {
		if v1beta1CRDExist(crds, c.Name, c.Version) {
			conflict = false
			break
		}
	}

	for _, c := range exists {
		if !v1beta1CRDExist(crds, c.Name, c.Version) {
			log.Printf("crd %s is needed", c.Name)
			exist = false
			break
		}
	}
	return conflict, exist
}

func doCheckV1(crds []v1.CustomResourceDefinition, conflicts, exists []CRD) (bool, bool) {
	conflict := true
	exist := true
	for _, c := range conflicts {
		if v1CRDExist(crds, c.Name, c.Version) {
			conflict = false
			break
		}
	}

	for _, c := range exists {
		if !v1CRDExist(crds, c.Name, c.Version) {
			log.Printf("crd %s is needed", c.Name)
			exist = false
			break
		}
	}
	return conflict, exist
}

func v1CRDExist(crds []v1.CustomResourceDefinition, name, version string) bool {
	for _, i := range crds {
		if i.Name == name {
			established := false
			namesAccepted := false
			for _, c := range i.Status.Conditions {
				if c.Type == v1.NamesAccepted && c.Status == v1.ConditionTrue {
					log.Printf("crd %s names accepted", name)
					namesAccepted = true
				}
				if c.Type == v1.Established && c.Status == v1.ConditionTrue {
					log.Printf("crd %s established", name)
					established = true
				}
			}
			if !established || !namesAccepted {
				return false
			}
			if len(version) == 0 {
				return true
			} else {
				for _, v := range i.Spec.Versions {
					if v.Name == version && v.Served {
						return true
					}
				}
			}

		}
	}
	return false
}

func v1beta1CRDExist(crds []v1beta1.CustomResourceDefinition, name, version string) bool {
	for _, i := range crds {
		if i.Name == name {
			established := false
			namesAccepted := false
			for _, c := range i.Status.Conditions {
				if c.Type == v1beta1.Established && c.Status == v1beta1.ConditionTrue {
					log.Printf("crd %s established", name)
					log.Printf("established time %s", c.LastTransitionTime)
					established = true
				}
				if c.Type == v1beta1.NamesAccepted && c.Status == v1beta1.ConditionTrue {
					log.Printf("crd %s names accepted", name)
					log.Printf("accepted time %s", c.LastTransitionTime)
					namesAccepted = true
				}
			}
			if !established || !namesAccepted {
				return false
			}
			log.Printf("crd create time %v", i.CreationTimestamp)
			if len(version) == 0 {
				return true
			} else if version == i.Spec.Version {
				return true
			}
		}
	}
	return false
}
