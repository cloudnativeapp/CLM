module cloudnativeapp/clm

go 1.13

require (
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/gofrs/flock v0.8.0
	github.com/jonboulle/clockwork v0.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.13.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.4
	k8s.io/cli-runtime v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/component-base v0.19.4
	k8s.io/klog/v2 v2.4.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kubectl v0.19.4
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)
