module kubevirt.io/kubevirt

require (
	cloud.google.com/go v0.0.0-20160913182117-3b1ae45394a2 // indirect
	github.com/Azure/go-autorest v11.1.0+incompatible // indirect
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-iptables v0.4.1
	github.com/dgrijalva/jwt-go v0.0.0-20160705203006-01aeca54ebda // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/emicklei/go-restful v2.6.0+incompatible
	github.com/emicklei/go-restful-openapi v0.10.0
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/fatih/color v1.7.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8 // indirect
	github.com/go-kit/kit v0.8.0
	github.com/go-openapi/analysis v0.17.2 // indirect
	github.com/go-openapi/errors v0.17.2
	github.com/go-openapi/loads v0.17.2 // indirect
	github.com/go-openapi/runtime v0.17.2 // indirect
	github.com/go-openapi/spec v0.17.2
	github.com/go-openapi/strfmt v0.17.0
	github.com/go-openapi/validate v0.18.0
	github.com/gogo/protobuf v1.1.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v0.0.0-20190509174753-d74b93584564
	github.com/golang/protobuf v1.3.1
	github.com/google/goexpect v0.0.0-20190425035906-112704a48083
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/google/goterm v0.0.0-20190311235235-ce302be1d114 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20180330165814-781450b3c4fc // indirect
	github.com/gorilla/websocket v0.0.0-20180228210902-0647012449a1
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20181121151021-386d141f4c94
	github.com/krolaw/dhcp4 v0.0.0-20180925202202-7cead472c414
	github.com/libvirt/libvirt-go v5.0.0+incompatible
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-runewidth v0.0.0-20181218000649-703b5e6b11ae // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.1-0.20190515112211-6a48b4839f85
	github.com/openshift/api v3.9.1-0.20190401220125-3a6077f1f910+incompatible
	github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	github.com/operator-framework/go-appr v0.0.0-20180917210448-f2aef88446f2 // indirect
	github.com/operator-framework/operator-marketplace v0.0.0-20190508022032-93d436f211c1
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v0.9.3
	github.com/smartystreets/goconvey v0.0.0-20190330032615-68dc04aab96a // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.1
	github.com/subgraph/libmacouflage v0.0.1
	github.com/vishvananda/netlink v0.0.0-20180206203732-d35d6b58e1cb
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc // indirect
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	google.golang.org/genproto v0.0.0-20170731182057-09f6ed296fc6 // indirect
	google.golang.org/grpc v1.13.0
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/ini.v1 v1.42.0
	gopkg.in/yaml.v2 v2.2.1
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-aggregator v0.0.0-20190228175259-3e0149950b0e
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a
	kubevirt.io/client-go v0.0.0-00010101000000-000000000000
	kubevirt.io/containerized-data-importer v1.8.1-0.20190516083534-83c12eaae2ed
	kubevirt.io/qe-tools v0.1.3-0.20190512140058-934db0579e0c
	sigs.k8s.io/controller-runtime v0.1.9 // indirect
)

replace github.com/k8snetworkplumbingwg/network-attachment-definition-client => github.com/booxter/network-attachment-definition-client v0.0.0-20181121221720-d76adb95b0b7

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4

replace k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628

replace github.com/go-kit/kit => github.com/go-kit/kit v0.3.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v11.0.0+incompatible

replace kubevirt.io/containerized-data-importer => kubevirt.io/containerized-data-importer v1.8.1-0.20190530153240-6734c225525a

replace github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a

replace kubevirt.io/client-go => ./staging/src/kubevirt.io/client-go
