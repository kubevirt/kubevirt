module kubevirt.io/kubevirt

require (
	github.com/Azure/go-autorest/autorest v0.9.1 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.6.0 // indirect
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-iptables v0.4.3
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/coreos/prometheus-operator v0.31.1
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/emicklei/go-restful-openapi v0.10.0
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/fatih/color v1.7.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/errors v0.19.2
	github.com/go-openapi/runtime v0.17.2 // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/go-openapi/strfmt v0.19.0
	github.com/go-openapi/validate v0.18.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.1.1
	github.com/golang/protobuf v1.3.1
	github.com/google/goexpect v0.0.0-20190425035906-112704a48083
	github.com/google/gofuzz v1.0.0
	github.com/google/goterm v0.0.0-20190311235235-ce302be1d114 // indirect
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20190920090233-ccc72ee9eb57
	github.com/krolaw/dhcp4 v0.0.0-20180925202202-7cead472c414
	github.com/libvirt/libvirt-go v5.0.0+incompatible
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-runewidth v0.0.0-20181218000649-703b5e6b11ae // indirect
	github.com/mfranczy/crd-rest-coverage v0.1.0
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.1-0.20190515112211-6a48b4839f85
	github.com/openshift/api v3.9.1-0.20190401220125-3a6077f1f910+incompatible
	github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	github.com/operator-framework/go-appr v0.0.0-20180917210448-f2aef88446f2 // indirect
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190128024246-5eb7ae5bdb7a
	github.com/operator-framework/operator-marketplace v0.0.0-20190508022032-93d436f211c1
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v0.9.3
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/subgraph/libmacouflage v0.0.1
	github.com/vishvananda/netlink v0.0.0-20180206203732-d35d6b58e1cb
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980
	golang.org/x/oauth2 v0.0.0-20181105165119-ca4130e427c7 // indirect
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f
	google.golang.org/grpc v1.19.1
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/ini.v1 v1.42.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190725062911-6607c48751ae
	k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed
	k8s.io/apimachinery v0.0.0-20190719140911-bfcf53abc9f8
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20190228175259-3e0149950b0e
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a
	kubevirt.io/client-go v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer v1.10.6
	kubevirt.io/qe-tools v0.1.3-0.20190512140058-934db0579e0c
	sigs.k8s.io/controller-runtime v0.1.9 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v11.0.0+incompatible
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.0.0-20181206002233-dd6f23e7207c
	github.com/go-kit/kit => github.com/go-kit/kit v0.3.0
	github.com/golang/glog => ./staging/src/github.com/golang/glog
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	kubevirt.io/client-go => ./staging/src/kubevirt.io/client-go
)
