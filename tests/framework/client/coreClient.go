package client

import (
	"sync"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// TestCoreV1Client is used to interact with features provided by the  group.
type TestCoreV1Client struct {
	corev1.CoreV1Client
}

var (
	namespaceOnce             sync.Once
	namespaceInterface        corev1.NamespaceInterface
	nodeOnce                  sync.Once
	nodeInterface             corev1.NodeInterface
	persistentVolumeOnce      sync.Once
	persistentVolumeInterface corev1.PersistentVolumeInterface
)

func (c *TestCoreV1Client) Namespaces() corev1.NamespaceInterface {
	namespaceOnce.Do(func() {
		namespaceInterface = newNamespaces(c)
		resourcesToClean = append(resourcesToClean, namespaceInterface.(CleanableResource))
	})
	return namespaceInterface
}

func (c *TestCoreV1Client) Nodes() corev1.NodeInterface {
	nodeOnce.Do(func() {
		nodeInterface = newNodes(c)
		resourcesToClean = append(resourcesToClean, nodeInterface.(CleanableResource))
	})
	return nodeInterface
}

func (c *TestCoreV1Client) PersistentVolumes() corev1.PersistentVolumeInterface {
	persistentVolumeOnce.Do(func() {
		persistentVolumeInterface = newPersistentVolumes(c)
		resourcesToClean = append(resourcesToClean, persistentVolumeInterface.(CleanableResource))
	})
	return persistentVolumeInterface
}

// CoreNewForConfig creates a new CoreV1Client for the given RESTClient.
func CoreNewForConfig(c *rest.Config) *TestCoreV1Client {
	coreClient := corev1.NewForConfigOrDie(c)
	return &TestCoreV1Client{*coreClient}
}
