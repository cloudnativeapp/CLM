package cliruntime

import (
	"cloudnativeapp/clm/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

type GetOption struct {
	AllNamespaces  bool
	Namespace      string
	IgnoreNotFound bool
	Builder        *resource.Builder
}

func NewGetOption(ignoreNotFound bool, namespace string, allNamespace bool) *GetOption {
	g := &GetOption{}
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	g.Builder = resource.NewBuilder(configFlags)
	g.Namespace = namespace
	g.IgnoreNotFound = ignoreNotFound
	g.AllNamespaces = allNamespace
	return g
}

func (g *GetOption) Run(args []string) (int, error) {
	cLog.V(utils.Debug).Info("start get")
	r := g.Builder.
		Unstructured().
		NamespaceParam(g.Namespace).DefaultNamespace().AllNamespaces(g.AllNamespaces).
		ResourceTypeOrNameArgs(true, args...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if g.IgnoreNotFound {
		r.IgnoreErrors(apierrors.IsNotFound)
	}
	if err := r.Err(); err != nil {
		return 0, err
	}

	rs, err := r.Infos()
	if err != nil {
		return 0, err
	}
	cLog.V(utils.Debug).Info("resource got", "result", rs)

	return len(rs), nil
}
