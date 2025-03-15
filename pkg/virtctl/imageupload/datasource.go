package imageupload

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (c *command) handleDataSource() error {
	ds, err := c.client.CdiClient().CdiV1beta1().DataSources(c.namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err == nil {
		return c.updateExistingDataSource(ds)
	}

	if k8serrors.IsNotFound(err) {
		return c.createNewDataSource()
	}

	return err
}

func (c *command) createNewDataSource() error {
	ds := &cdiv1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
			Labels:    map[string]string{},
		},
		Spec: cdiv1.DataSourceSpec{
			Source: cdiv1.DataSourceSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      c.name,
					Namespace: c.namespace,
				},
			},
		},
	}
	c.setDefaultInstancetypeLabels(&ds.ObjectMeta)

	_, err := c.client.CdiClient().CdiV1beta1().DataSources(c.namespace).Create(context.Background(), ds, metav1.CreateOptions{})
	if err == nil {
		c.cmd.Printf("Created a new DataSource %s/%s\n", c.namespace, c.name)
	}
	return err
}

func (c *command) updateExistingDataSource(ds *cdiv1.DataSource) error {
	c.setDefaultInstancetypeLabels(&ds.ObjectMeta)

	patchBytes, err := patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/metadata/labels",
			Value: ds.Labels,
		},
		patch.PatchOperation{
			Op:   patch.PatchReplaceOp,
			Path: "/spec/source/pvc",
			Value: map[string]string{
				"name":      c.name,
				"namespace": c.namespace,
			},
		},
	)
	if err != nil {
		return err
	}

	if _, err = c.client.CdiClient().CdiV1beta1().DataSources(ds.Namespace).Patch(context.Background(), ds.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err == nil {
		c.cmd.Printf("Updated an existing DataSource %s/%s\n", ds.Namespace, ds.Name)
	}
	return err
}
