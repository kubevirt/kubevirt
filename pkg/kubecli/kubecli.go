package kubecli

import (
	"flag"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/clientcmd"
	"kubevirt/core/pkg/api/v1"
)

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
}

func Get() (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	return kubernetes.NewForConfig(config)
}

func GetRESTCLient() (*rest.RESTClient, error) {

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	return rest.RESTClientFor(config)
}
