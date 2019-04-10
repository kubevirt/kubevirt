package validatingwebhook

import (
	"sync"

	"k8s.io/client-go/kubernetes"
)

var kubeClient kubernetes.Interface
var once sync.Once

// GetClient returns kubernetes client
func GetClient() kubernetes.Interface {
	return kubeClient
}

// SetClient sets kubernetes
func SetClient(client kubernetes.Interface) {
	once.Do(func() {
		kubeClient = client
	})
}
