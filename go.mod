module kubevirt.io/kubevirt

require (
	github.com/Azure/go-autorest/autorest v0.9.1 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.6.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/blang/semver v3.5.1+incompatible
	github.com/containernetworking/plugins v0.8.2
	github.com/coreos/go-iptables v0.4.3
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/prometheus-operator v0.35.0
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/emicklei/go-restful v2.10.0+incompatible
	github.com/emicklei/go-restful-openapi v0.10.0
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fatih/color v1.9.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/errors v0.19.2
	github.com/go-openapi/spec v0.19.3
	github.com/go-openapi/strfmt v0.19.0
	github.com/go-openapi/validate v0.19.2
	github.com/gogo/protobuf v1.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.4.1
	github.com/google/go-github/v32 v32.0.0
	github.com/google/goexpect v0.0.0-20190425035906-112704a48083
	github.com/google/gofuzz v1.0.0
	github.com/google/goterm v0.0.0-20190311235235-ce302be1d114 // indirect
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20191119172530-79f836b90111
	github.com/krolaw/dhcp4 v0.0.0-20180925202202-7cead472c414
	github.com/kubernetes-csi/external-snapshotter/v2 v2.0.1
	github.com/mattn/go-runewidth v0.0.0-20181218000649-703b5e6b11ae // indirect
	github.com/mfranczy/crd-rest-coverage v0.1.0
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/selinux v1.6.0
	github.com/openshift/api v0.0.0-20191219222812-2987a591a72c
	github.com/openshift/client-go v0.0.0-20191125132246-f6563a70e19a
	github.com/operator-framework/go-appr v0.0.0-20180917210448-f2aef88446f2 // indirect
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190128024246-5eb7ae5bdb7a
	github.com/operator-framework/operator-marketplace v0.0.0-20190508022032-93d436f211c1
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v1.1.0
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/subgraph/libmacouflage v0.0.1
	github.com/vishvananda/netlink v0.0.0-20181108222139-023a6dafdcdf
	golang.org/x/crypto v0.0.0-20190927123631-a832865fa7ad
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/ini.v1 v1.42.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery v0.17.1-beta.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.16.4
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	kubevirt.io/client-go v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer v1.10.9
	kubevirt.io/qe-tools v0.1.6
	libvirt.org/libvirt-go v6.5.0+incompatible
	sigs.k8s.io/controller-runtime v0.1.9 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v11.0.0+incompatible
	github.com/go-kit/kit => github.com/go-kit/kit v0.3.0
	github.com/golang/glog => ./staging/src/github.com/golang/glog
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191219222812-2987a591a72c
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191125132246-f6563a70e19a
	github.com/operator-framework/operator-lifecycle-manager => github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190128024246-5eb7ae5bdb7a

	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/apiserver => k8s.io/apiserver v0.16.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.4
	k8s.io/code-generator => k8s.io/code-generator v0.16.4
	k8s.io/component-base => k8s.io/component-base v0.16.4
	k8s.io/cri-api => k8s.io/cri-api v0.16.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.4
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.4
	k8s.io/kubectl => k8s.io/kubectl v0.16.4
	k8s.io/kubelet => k8s.io/kubelet v0.16.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.4
	k8s.io/metrics => k8s.io/metrics v0.16.4
	k8s.io/node-api => k8s.io/node-api v0.16.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.16.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.4

	kubevirt.io/client-go => ./staging/src/kubevirt.io/client-go
)

go 1.13
