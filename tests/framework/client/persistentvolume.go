package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var createdPVs = make(map[string]context.Context, 0)

// persistentVolumes implements PersistentVolumeInterface
type persistentVolumes struct {
	v12.PersistentVolumeInterface
}

// newPersistentVolumes returns a PersistentVolumes
func newPersistentVolumes(c *TestCoreV1Client) *persistentVolumes {
	return &persistentVolumes{
		c.CoreV1Client.PersistentVolumes(),
	}
}

func (c *persistentVolumes) Clean() {
	for name, ctx := range createdPVs {
		err := c.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			panic(err)
		}
	}
}

func (c *persistentVolumes) Create(ctx context.Context, persistentVolume *v1.PersistentVolume, opts metav1.CreateOptions) (result *v1.PersistentVolume, err error) {
	created, err := c.PersistentVolumeInterface.Create(ctx, persistentVolume, opts)
	if err == nil && opts.DryRun == nil {
		createdPVs[created.Name] = ctx
	}

	return created, err
}

func (c *persistentVolumes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	err := c.PersistentVolumeInterface.Delete(ctx, name, opts)
	if _, exist := createdPVs[name]; exist && err == nil && opts.DryRun == nil {
		delete(createdPVs, name)
	}

	return err
}
