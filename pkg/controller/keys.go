package controller

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

func NamespacedKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
