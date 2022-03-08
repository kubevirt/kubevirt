package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var createdNamespaces = make(map[string]context.Context, 0)

// namespaces implements NamespaceInterface
type namespaces struct {
	v12.NamespaceInterface
}

// newNamespaces returns a Namespaces
func newNamespaces(c *TestCoreV1Client) *namespaces {
	return &namespaces{
		c.CoreV1Client.Namespaces(),
	}
}

func (c *namespaces) Clean() {
	for name, ctx := range createdNamespaces {
		err := c.NamespaceInterface.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			panic(err)
		}
	}
}

func (c *namespaces) Create(ctx context.Context, namespace *v1.Namespace, opts metav1.CreateOptions) (result *v1.Namespace, err error) {
	created, err := c.NamespaceInterface.Create(ctx, namespace, opts)
	if err == nil && opts.DryRun == nil {
		createdNamespaces[created.Name] = ctx
	}

	return created, err
}

func (c *namespaces) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	err := c.NamespaceInterface.Delete(ctx, name, opts)
	if _, exist := createdNamespaces[name]; exist && err == nil && opts.DryRun == nil {
		delete(createdNamespaces, name)
	}

	return err
}
