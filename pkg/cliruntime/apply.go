package cliruntime

import (
	"cloudnativeapp/clm/pkg/utils"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

type ApplyOptions struct {
	Builder           *resource.Builder
	objects           []*resource.Info
	Selector          string
	VisitedNamespaces sets.String
	VisitedUids       sets.String
	Urls              []string
	YamlStr           string
	IgnoreError       bool
}

const (
	kubectlPrefix = "kubectl.kubernetes.io/"

	LastAppliedConfigAnnotation = kubectlPrefix + "last-applied-configuration"
)

var cLog = ctrl.Log.WithName("cli-runtime")

var metadataAccessor = meta.NewAccessor()

func NewApplyOptions(urls []string, yamlStr string, ignoreError bool) *ApplyOptions {
	n := &ApplyOptions{}
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	n.Builder = resource.NewBuilder(configFlags)
	n.Urls = urls
	n.YamlStr = yamlStr
	n.VisitedNamespaces = sets.NewString()
	n.VisitedUids = sets.NewString()
	n.IgnoreError = ignoreError
	return n
}

func (n *ApplyOptions) GetObjects() ([]*resource.Info, error) {
	cLog.V(utils.Debug).Info("try to get objects from stream")
	var err error = nil
	var result *resource.Result
	b := n.Builder.
		Unstructured().
		ContinueOnError().
		DefaultNamespace().
		LabelSelectorParam(n.Selector).
		Flatten()
	if len(n.YamlStr) != 0 {
		b.Stream(strings.NewReader(n.YamlStr), "test")
	}

	for _, s := range n.Urls {
		if strings.Index(s, "http://") == 0 || strings.Index(s, "https://") == 0 {
			url, err := url.Parse(s)
			if err != nil {
				cLog.Error(err, fmt.Sprintf("the URL passed to filename %q is not valid", s))
				return nil, err
			}
			b.URL(3, url)
		}
	}

	result = b.Do()
	n.objects, err = result.Infos()

	return n.objects, err
}

func (n *ApplyOptions) Run() error {
	cLog.V(utils.Debug).Info("start apply")
	errs := []error{}
	infos, err := n.GetObjects()
	if err != nil {
		errs = append(errs, err)
	}
	if len(infos) == 0 && len(errs) == 0 {
		return fmt.Errorf("no objects passed to apply")
	}
	// Iterate through all objects, applying each one.
	for _, info := range infos {
		cLog.V(utils.Info).Info(fmt.Sprintf("apply %s", info.Name))
		if err := n.applyOneObject(info); err != nil {
			errs = append(errs, err)
		}
	}

	// If any errors occurred during apply, then return error (or
	// aggregate of errors).
	if len(errs) == 1 {
		return errs[0]
	}
	if len(errs) > 1 {
		return utilerrors.NewAggregate(errs)
	}
	return nil
}

func (n *ApplyOptions) MarkNamespaceVisited(info *resource.Info) {
	if info.Namespaced() {
		n.VisitedNamespaces.Insert(info.Namespace)
	}
}

func (n *ApplyOptions) MarkObjectVisited(info *resource.Info) error {
	metadata, err := meta.Accessor(info.Object)
	if err != nil {
		return err
	}
	n.VisitedUids.Insert(string(metadata.GetUID()))
	return nil
}

func (n *ApplyOptions) applyOneObject(info *resource.Info) error {
	cLog.V(utils.Debug).Info("start apply one object", "info", info.Name)
	n.MarkNamespaceVisited(info)
	helper := resource.NewHelper(info.Client, info.Mapping)

	modified, err := GetModifiedConfiguration(info.Object, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving modified configuration from:\n%s\nfor:", info.String()), info.Source, err)
	}

	if err := info.Get(); err != nil {
		if !errors.IsNotFound(err) {
			return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving current configuration of:\n%s\nfrom server for:", info.String()), info.Source, err)
		}
		// Create the resource if it doesn't exist
		// First, update the annotation used by kubectl apply
		if err := CreateApplyAnnotation(info.Object, unstructured.UnstructuredJSONScheme); err != nil {
			return cmdutil.AddSourceToErr("creating", info.Source, err)
		}
		// Then create the resource and skip the three-way merge
		obj, err := helper.Create(info.Namespace, true, info.Object)
		if err != nil {
			return cmdutil.AddSourceToErr("creating", info.Source, err)
		}
		info.Refresh(obj, n.IgnoreError)

		if err := n.MarkObjectVisited(info); err != nil {
			return err
		}
		//todo wait resource ready
		return nil
	}

	if err := n.MarkObjectVisited(info); err != nil {
		return err
	}

	metadata, _ := meta.Accessor(info.Object)
	annotationMap := metadata.GetAnnotations()
	if _, ok := annotationMap[LastAppliedConfigAnnotation]; !ok {
		cLog.V(utils.Warn).Info("annotationMap err")
	}

	patcher, err := newPatcher(info, helper)
	if err != nil {
		return err
	}
	patchBytes, patchedObject, err := patcher.Patch(info.Object, modified, info.Source, info.Namespace, info.Name, nil)
	if err != nil {
		return cmdutil.AddSourceToErr(fmt.Sprintf("applying patch:\n%s\nto:\n%v\nfor:", patchBytes, info), info.Source, err)
	}

	info.Refresh(patchedObject, true)
	return nil
}

func GetModifiedConfiguration(obj runtime.Object, annotate bool, codec runtime.Encoder) ([]byte, error) {
	var modified []byte
	annots, err := metadataAccessor.Annotations(obj)
	if err != nil {
		return nil, err
	}

	if annots == nil {
		annots = map[string]string{}
	}

	original := annots[LastAppliedConfigAnnotation]
	delete(annots, LastAppliedConfigAnnotation)
	if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
		return nil, err
	}

	modified, err = runtime.Encode(codec, obj)
	if err != nil {
		return nil, err
	}

	if annotate {
		annots[LastAppliedConfigAnnotation] = string(modified)
		if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
			return nil, err
		}

		modified, err = runtime.Encode(codec, obj)
		if err != nil {
			return nil, err
		}
	}

	annots[LastAppliedConfigAnnotation] = original
	if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
		return nil, err
	}

	return modified, nil
}

func CreateApplyAnnotation(obj runtime.Object, codec runtime.Encoder) error {
	modified, err := GetModifiedConfiguration(obj, false, codec)
	if err != nil {
		return err
	}
	return setOriginalConfiguration(obj, modified)
}

func setOriginalConfiguration(obj runtime.Object, original []byte) error {
	if len(original) < 1 {
		return nil
	}

	annots, err := metadataAccessor.Annotations(obj)
	if err != nil {
		return err
	}

	if annots == nil {
		annots = map[string]string{}
	}

	annots[LastAppliedConfigAnnotation] = string(original)
	return metadataAccessor.SetAnnotations(obj, annots)
}
