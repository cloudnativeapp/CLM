package cliruntime

import (
	"cloudnativeapp/clm/pkg/utils"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	cmdwait "k8s.io/kubectl/pkg/cmd/wait"
	"net/url"
	"strings"
)

const (
	CRD_NOT_EXIST_ERROR = "doesn't have a resource type"
)

type DeleteOptions struct {
	Builder     *resource.Builder
	GracePeriod int
	Result      *resource.Result
}

func NewDeleteOptions(urlsInput []string, yamlStr string) (*DeleteOptions, error) {
	d := &DeleteOptions{}
	d.GracePeriod = 1
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	d.Builder = resource.NewBuilder(configFlags)
	r := d.Builder.
		Unstructured().
		ContinueOnError().
		DefaultNamespace().
		SelectAllParam(false).
		AllNamespaces(false).
		RequireObject(false).
		Flatten()
	if len(yamlStr) != 0 {
		r.Stream(strings.NewReader(yamlStr), "test")
	}

	for _, s := range urlsInput {
		if strings.Index(s, "http://") == 0 || strings.Index(s, "https://") == 0 {
			url, err := url.Parse(s)
			if err != nil {
				cLog.V(utils.Warn).Info("the URL passed to filename is not valid",
					"filename", s, "error", err.Error())
				continue
			}
			r.URL(3, url)
		}
	}

	result := r.Do()
	err := result.Err()
	if err != nil {
		return nil, err
	}
	d.Result = result
	return d, nil
}

func (o *DeleteOptions) RunDelete() error {
	cLog.V(utils.Debug).Info("start delete")
	return o.DeleteResult(o.Result)
}

func (o *DeleteOptions) DeleteResult(r *resource.Result) error {
	found := 0
	uidMap := cmdwait.UIDMap{}
	err := r.Visit(func(info *resource.Info, err error) error {
		cLog.V(utils.Debug).Info("start delete an info", "info", info.Name)
		if err != nil {
			return err
		}
		found++

		if err := validateCRDStatus(info); err != nil {
			return err
		}

		options := &metav1.DeleteOptions{}
		if o.GracePeriod >= 0 {
			options = metav1.NewDeleteOptions(int64(o.GracePeriod))
		}
		policy := metav1.DeletePropagationBackground
		options.PropagationPolicy = &policy
		response, err := o.deleteResource(info, options)
		if err != nil {
			return err
		}
		resourceLocation := cmdwait.ResourceLocation{
			GroupResource: info.Mapping.Resource.GroupResource(),
			Namespace:     info.Namespace,
			Name:          info.Name,
		}
		if status, ok := response.(*metav1.Status); ok && status.Details != nil {
			uidMap[resourceLocation] = status.Details.UID
			return nil
		}
		responseMetadata, err := meta.Accessor(response)
		if err != nil {
			// we don't have UID, but we didn't fail the delete, next best thing is just skipping the UID
			klog.V(1).Info(err)
			return nil
		}
		uidMap[resourceLocation] = responseMetadata.GetUID()

		return nil
	})
	if err != nil {
		return err
	}
	if found == 0 {
		cLog.V(utils.Warn).Info("No resources found")
		return nil
	}

	return nil
}

func validateCRDStatus(info *resource.Info) error {
	gvk := info.Object.GetObjectKind()
	if gvk != nil && strings.ToLower(gvk.GroupVersionKind().Kind) == "customresourcedefinition" {
		if count, err := CheckCRNum(info.Name); err != nil {
			return err
		} else if count == 0 {
			return nil
		} else {
			return errors.New(fmt.Sprintf("can not delete CRD %s with CR.", gvk.GroupVersionKind().Group))
		}
	}
	return nil
}

func CheckCRNum(kind string) (int, error) {
	g := NewGetOption(true, "", true)
	count, err := g.Run([]string{kind})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), CRD_NOT_EXIST_ERROR) {
			return 0, nil
		}
		cLog.V(utils.Debug).Info("find resource error", "error", err.Error())
		return 0, err
	}
	return count, nil
}

func (o *DeleteOptions) deleteResource(info *resource.Info, deleteOptions *metav1.DeleteOptions) (runtime.Object, error) {
	deleteResponse, err := resource.
		NewHelper(info.Client, info.Mapping).
		DeleteWithOptions(info.Namespace, info.Name, deleteOptions)
	if err != nil {
		return nil, cmdutil.AddSourceToErr("deleting", info.Source, err)
	}

	return deleteResponse, nil
}
