module github.com/kubevirt/hyperconverged-cluster-operator

go 1.13

require (
	cloud.google.com/go v0.37.4 // indirect
	github.com/MarSik/kubevirt-ssp-operator v1.0.20
	github.com/appscode/jsonpatch v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/prometheus-operator v0.35.0 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/golang/mock v1.3.1 // indirect
	github.com/imdario/mergo v0.3.8
	github.com/kubevirt/cluster-network-addons-operator v0.3.1-0.20191002163030-b0275809cef4
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/openshift/client-go v0.0.0-20190923180330-3b6373338c9b // indirect
	github.com/openshift/custom-resource-status v0.0.0-20190822192428-e62f2f3b79f3
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20191220211133-23f5c0292434
	github.com/operator-framework/operator-sdk v0.10.0
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20190621222207-cc06ce4a13d4 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55 // indirect
	k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apimachinery v0.15.7
	k8s.io/apiserver v0.0.0 // indirect
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v0.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	kubevirt.io/client-go v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer v1.11.0
	kubevirt.io/kubevirt v0.23.3
	sigs.k8s.io/controller-runtime v0.3.1-0.20191016212439-2df793d02076
	sigs.k8s.io/testing_frameworks v0.1.2 // indirect

)

// Pinned to kubernetes-1.13.4 to align with
// https://github.com/operator-framework/operator-sdk/blob/v0.10.x/internal/pkg/scaffold/go_mod.go
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190409021813-1ec86e4da56c
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190409023024-d644b00f3b79
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190409023614-027c502bb854
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190409021516-bd2732e5c3f7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190409022021-00b8e31abe9d
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
	k8s.io/kubectl => k8s.io/kubectl v0.15.7
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.1
)

replace (
	github.com/appscode/jsonpatch => github.com/appscode/jsonpatch v1.0.1
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	github.com/go-kit/kit => github.com/go-kit/kit v0.3.0
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20191025120018-fb3724fc7bdf
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.10.0
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20190424153033-d3245f150225
	kubevirt.io/client-go => kubevirt.io/client-go v0.23.3
	kubevirt.io/kubevirt => kubevirt.io/kubevirt v0.23.3
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
