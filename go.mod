module kubevirt.io/kubevirt

require (
	github.com/Masterminds/semver v1.5.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/c9s/goprocinfo v0.0.0-20210130143923-c95fcf8c64a8
	github.com/containernetworking/plugins v0.9.1
	github.com/coreos/go-iptables v0.5.0
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/prometheus-operator v0.38.1-0.20200424145508-7e176fda06cc
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/emicklei/go-restful-openapi v1.2.0
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/errors v0.19.9
	github.com/go-openapi/spec v0.20.3
	github.com/go-openapi/strfmt v0.20.0
	github.com/go-openapi/validate v0.20.2
	github.com/gogo/protobuf v1.3.2
	github.com/golang/glog v1.0.0
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-github/v32 v32.0.0
	github.com/google/goexpect v0.0.0-20190425035906-112704a48083
	github.com/google/gofuzz v1.1.0
	github.com/google/uuid v1.3.0
	github.com/gordonklaus/ineffassign v0.0.0-20210209182638-d0e41b2fc8ed
	github.com/gorilla/websocket v1.4.2
	github.com/imdario/mergo v0.3.11
	github.com/insomniacslk/dhcp v0.0.0-20201112113307-4de412bc85d8
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20191119172530-79f836b90111
	github.com/krolaw/dhcp4 v0.0.0-20180925202202-7cead472c414
	github.com/kubernetes-csi/external-snapshotter/v2 v2.1.1
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/mitchellh/go-vnc v0.0.0-20150629162542-723ed9867aed
	github.com/moby/sys/mountinfo v0.4.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.3
	github.com/opencontainers/runc v1.0.0
	github.com/opencontainers/selinux v1.8.2
	github.com/openshift/api v0.0.0
	github.com/openshift/client-go v0.0.0
	github.com/openshift/library-go v0.0.0-20210205203934-9eb0d970f2f4
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190725173916-b56e63a643cc
	github.com/operator-framework/operator-marketplace v0.0.0-20190617165322-1cbd32624349
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.28.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/subgraph/libmacouflage v0.0.1
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/wadey/gocovmerge v0.0.0-20160331181800-b5bfa59ec0ad
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/net v0.0.0-20211209124913-491a49abca63
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	google.golang.org/grpc v1.40.0
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.11
	k8s.io/apiextensions-apiserver v0.21.11
	k8s.io/apimachinery v0.21.11
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.21.11
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	kubevirt.io/api v0.0.0-00010101000000-000000000000
	kubevirt.io/client-go v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer v1.41.0
	kubevirt.io/containerized-data-importer-api v1.41.0
	kubevirt.io/controller-lifecycle-operator-sdk v0.2.1
	kubevirt.io/qe-tools v0.1.6
	libvirt.org/go/libvirt v1.8006.0
	mvdan.cc/sh/v3 v3.1.1
	sigs.k8s.io/yaml v1.2.0
)

require (
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/willf/bitset v1.1.11 // indirect
	golang.org/x/tools v0.1.6-0.20210820212750-d4cc65f0b2ff // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
)

require (
	github.com/google/goterm v0.0.0-20190311235235-ce302be1d114 // indirect; indirect github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/operator-framework/go-appr v0.0.0-20180917210448-f2aef88446f2 // indirect
)

replace (
	github.com/golang/glog => ./staging/src/github.com/golang/glog
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega => github.com/onsi/gomega v1.10.1
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc92
	github.com/opencontainers/selinux => github.com/opencontainers/selinux v1.6.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191219222812-2987a591a72c
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/operator-lifecycle-manager => github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190128024246-5eb7ae5bdb7a

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

	kubevirt.io/api => ./staging/src/kubevirt.io/api
	kubevirt.io/client-go => ./staging/src/kubevirt.io/client-go

	kubevirt.io/containerized-data-importer => kubevirt.io/containerized-data-importer v1.41.0
	kubevirt.io/containerized-data-importer-api => kubevirt.io/containerized-data-importer-api v1.41.0

	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.2
)

go 1.16
