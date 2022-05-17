module github.com/kubevirt/hyperconverged-cluster-operator

go 1.18

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/evanphx/json-patch v5.6.0+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v1.2.3
	github.com/google/uuid v1.3.0
	github.com/imdario/mergo v0.3.12
	github.com/kubevirt/cluster-network-addons-operator v0.76.0
	github.com/kubevirt/tekton-tasks-operator/api v0.3.0
	github.com/onsi/ginkgo/v2 v2.1.3
	github.com/onsi/gomega v1.19.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/custom-resource-status v1.1.2
	github.com/operator-framework/api v0.14.0
	github.com/operator-framework/operator-lib v0.10.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.55.1
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.1.10
	k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	kubevirt.io/api v0.51.0
	kubevirt.io/containerized-data-importer-api v1.49.0
	// Remove this once SSP will also consume v0.2.4
	kubevirt.io/controller-lifecycle-operator-sdk v0.2.3 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.2.4
	kubevirt.io/ssp-operator/api v0.15.0
	sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/controller-tools v0.8.0
)

require (
	github.com/medik8s/node-maintenance-operator v0.12.1
	kubevirt.io/node-maintenance-operator v0.11.3
)

require (
	cloud.google.com/go v0.97.0 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logr/zapr v1.2.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/component-base v0.23.5 // indirect
	k8s.io/klog/v2 v2.40.1 // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

exclude k8s.io/cluster-bootstrap v0.0.0

exclude k8s.io/api v0.0.0

exclude k8s.io/apiextensions-apiserver v0.0.0

exclude k8s.io/apimachinery v0.0.0

exclude k8s.io/apiserver v0.0.0

exclude k8s.io/code-generator v0.0.0

exclude k8s.io/component-base v0.0.0

exclude k8s.io/kube-aggregator v0.0.0

exclude k8s.io/cli-runtime v0.0.0

exclude k8s.io/kubectl v0.0.0

exclude k8s.io/client-go v2.0.0-alpha.0.0.20181121191925-a47917edff34+incompatible

exclude k8s.io/client-go v0.0.0

exclude k8s.io/cloud-provider v0.0.0

exclude k8s.io/cri-api v0.0.0

exclude k8s.io/csi-translation-lib v0.0.0

exclude k8s.io/kube-controller-manager v0.0.0

exclude k8s.io/kube-proxy v0.0.0

exclude k8s.io/kube-scheduler v0.0.0

exclude k8s.io/kubelet v0.0.0

exclude k8s.io/legacy-cloud-providers v0.0.0

exclude k8s.io/metrics v0.0.0

exclude k8s.io/sample-apiserver v0.0.0

// Pinned to v0.23.5
replace (
	k8s.io/api => k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.5
	k8s.io/apiserver => k8s.io/apiserver v0.23.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.5
	k8s.io/client-go => k8s.io/client-go v0.23.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.5
	k8s.io/code-generator => k8s.io/code-generator v0.23.5
	k8s.io/component-base => k8s.io/component-base v0.23.5
	k8s.io/cri-api => k8s.io/cri-api v0.23.5
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.5
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.5
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.5
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.5
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.5
	k8s.io/kubectl => k8s.io/kubectl v0.23.5
	k8s.io/kubelet => k8s.io/kubelet v0.23.5
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.5
	k8s.io/metrics => k8s.io/metrics v0.23.5
	k8s.io/node-api => k8s.io/node-api v0.23.5
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.5
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.5
	k8s.io/sample-controller => k8s.io/sample-controller v0.23.5
)

replace (
	github.com/appscode/jsonpatch => github.com/appscode/jsonpatch v1.0.1
	github.com/go-kit/kit => github.com/go-kit/kit v0.12.0
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20191025120018-fb3724fc7bdf
)

// Aligning with https://github.com/kubevirt/containerized-data-importer-api/blob/release-v1.41.1
replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20220315184754-d7c10d0b647e
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/library-go => github.com/mhenriks/library-go v0.0.0-20200804184258-4fc3a5379c7a
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.2
)

replace github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce // Required by Helm

replace vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787

replace bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d

// Fixes various security issues forcing newer versions of affected dependencies,
// prune the list once not explicitly required
replace (
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/crypto/ssh/terminal => golang.org/x/crypto/ssh/terminal v0.0.0-20201221181555-eec23a3978ad
)
