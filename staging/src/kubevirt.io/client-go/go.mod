module kubevirt.io/client-go

require (
	github.com/coreos/prometheus-operator v0.31.1
	github.com/go-kit/kit v0.8.0
	github.com/go-openapi/spec v0.19.2
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/golang/mock v1.1.1
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/websocket v0.0.0-20180228210902-0647012449a1
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20181121151021-386d141f4c94
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.1-0.20190515112211-6a48b4839f85
	github.com/openshift/api v3.9.1-0.20190401220125-3a6077f1f910+incompatible // indirect
	github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	github.com/openshift/custom-resource-status v0.0.0-20190822192428-e62f2f3b79f3 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/spf13/pflag v1.0.3
	google.golang.org/appengine v1.5.0 // indirect
	k8s.io/api v0.0.0-20190725062911-6607c48751ae
	k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed
	k8s.io/apimachinery v0.0.0-20190719140911-bfcf53abc9f8
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058
	kubevirt.io/containerized-data-importer v1.10.6
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v11.0.0+incompatible
	github.com/go-kit/kit => github.com/go-kit/kit v0.3.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client => github.com/booxter/network-attachment-definition-client v0.0.0-20181121221720-d76adb95b0b7
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.4
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	kubevirt.io/containerized-data-importer => kubevirt.io/containerized-data-importer v1.10.6
)

go 1.13
