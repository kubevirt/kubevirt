module kubevirt.io/client-go

go 1.22.0

toolchain go1.22.2

require (
	github.com/go-kit/kit v0.13.0
	github.com/golang/glog v1.1.0
	github.com/golang/mock v1.6.0
	github.com/google/gofuzz v1.2.0
	github.com/gorilla/websocket v1.5.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20191119172530-79f836b90111
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/onsi/ginkgo/v2 v2.17.2
	github.com/onsi/gomega v1.33.1
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.68.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.30.0
	k8s.io/apiextensions-apiserver v0.30.0
	k8s.io/apimachinery v0.30.0
	k8s.io/client-go v0.30.0
	k8s.io/kube-openapi v0.30.0
	k8s.io/utils v0.0.0-20240423183400-0849a56e8f22
	kubevirt.io/api v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer-api v1.57.0-alpha1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20240424215950-a892ee059fd6 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openshift/api v0.0.0-20230503133300-8bbcb7ca7183 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/oauth2 v0.12.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.0.0-20220329064328-f3cc58c6ed90 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210105115604-44119421ec6b

	k8s.io/api => k8s.io/api v0.30.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.30.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.30.0
	k8s.io/apiserver => k8s.io/apiserver v0.30.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.30.0
	k8s.io/client-go => k8s.io/client-go v0.30.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.30.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.30.0
	k8s.io/code-generator => k8s.io/code-generator v0.30.0
	k8s.io/component-base => k8s.io/component-base v0.30.0
	k8s.io/cri-api => k8s.io/cri-api v0.30.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.30.0
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.30.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.30.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20240430033511-f0e62f92d13f
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.30.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.30.0
	k8s.io/kubectl => k8s.io/kubectl v0.30.0
	k8s.io/kubelet => k8s.io/kubelet v0.30.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.30.0
	k8s.io/metrics => k8s.io/metrics v0.30.0
	k8s.io/node-api => k8s.io/node-api v0.30.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.30.0
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.30.0
	k8s.io/sample-controller => k8s.io/sample-controller v0.30.0

	kubevirt.io/api => ../api
)
