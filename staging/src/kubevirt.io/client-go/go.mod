module kubevirt.io/client-go

go 1.16

require (
	github.com/coreos/prometheus-operator v0.38.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/spec v0.19.5
	github.com/golang/glog v1.0.0
	github.com/golang/mock v1.5.0
	github.com/google/gofuzz v1.1.0
	github.com/gorilla/websocket v1.4.2
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20191119172530-79f836b90111
	github.com/kubernetes-csi/external-snapshotter/v2 v2.1.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/client-go v0.0.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.21.11
	k8s.io/apiextensions-apiserver v0.21.11
	k8s.io/apimachinery v0.21.11
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	kubevirt.io/api v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer-api v1.41.0
)

require (
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210105115604-44119421ec6b
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47

	k8s.io/api => k8s.io/api v0.21.11
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.11
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.11
	k8s.io/apiserver => k8s.io/apiserver v0.21.11
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.11
	k8s.io/client-go => k8s.io/client-go v0.21.11
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.11
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.11
	k8s.io/code-generator => k8s.io/code-generator v0.21.11
	k8s.io/component-base => k8s.io/component-base v0.21.11
	k8s.io/cri-api => k8s.io/cri-api v0.21.11
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.11
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.11
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.11
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.11
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.11
	k8s.io/kubectl => k8s.io/kubectl v0.21.11
	k8s.io/kubelet => k8s.io/kubelet v0.21.11
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.11
	k8s.io/metrics => k8s.io/metrics v0.21.11
	k8s.io/node-api => k8s.io/node-api v0.21.11
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.11
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.21.11
	k8s.io/sample-controller => k8s.io/sample-controller v0.21.11

	kubevirt.io/api => ../api
)
