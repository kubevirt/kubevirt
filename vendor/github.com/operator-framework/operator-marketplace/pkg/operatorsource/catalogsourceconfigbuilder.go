package operatorsource

import (
	"fmt"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatastoreLabel is the label used in a CatalogSourceConfig to indicate that
// the resulting CatalogSource acts as the datastore for the OperatorSource
// if it is set to "true".
const DatastoreLabel string = "opsrc-datastore"

// OpsrcOwnerNameLabel is the label used to mark ownership over resources
// that are owned by the OperatorSource. When this label is set, the reconciler
// should handle these resources when the OperatorSource is deleted.
const OpsrcOwnerNameLabel string = "opsrc-owner-name"

// OpsrcOwnerNamespaceLabel is the label used to mark ownership over resources
// that are owned by the OperatorSource. When this label is set, the reconciler
// should handle these resources when the OperatorSource is deleted.
const OpsrcOwnerNamespaceLabel string = "opsrc-owner-namespace"

// CatalogSourceConfigBuilder builds a new CatalogSourceConfig type object.
type CatalogSourceConfigBuilder struct {
	object marketplace.CatalogSourceConfig
}

// CatalogSourceConfig returns a prepared CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) CatalogSourceConfig() *marketplace.CatalogSourceConfig {
	return &b.object
}

// WithTypeMeta sets TypeMeta of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithTypeMeta() *CatalogSourceConfigBuilder {
	b.object.TypeMeta = metav1.TypeMeta{
		APIVersion: fmt.Sprintf("%s/%s",
			marketplace.SchemeGroupVersion.Group, marketplace.SchemeGroupVersion.Version),
		Kind: marketplace.CatalogSourceConfigKind,
	}

	return b
}

// WithNamespacedName sets name and namespace of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithNamespacedName(namespace, name string) *CatalogSourceConfigBuilder {
	b.object.SetNamespace(namespace)
	b.object.SetName(name)

	return b
}

// WithLabels sets appropriate labels for the CatalogSourceConfig object. It
// applies all labels associated with an OperatorSource object specified in
// opsrcLabels.
func (b *CatalogSourceConfigBuilder) WithLabels(opsrcLabels map[string]string) *CatalogSourceConfigBuilder {
	labels := map[string]string{
		DatastoreLabel: "true",
	}

	for key, value := range opsrcLabels {
		labels[key] = value
	}

	for key, value := range b.object.GetLabels() {
		labels[key] = value
	}

	b.object.SetLabels(labels)

	return b
}

// WithOwnerLabel sets the owner label of the CatalogSourceConfig object to the given owner.
func (b *CatalogSourceConfigBuilder) WithOwnerLabel(owner *marketplace.OperatorSource) *CatalogSourceConfigBuilder {
	labels := map[string]string{
		OpsrcOwnerNameLabel:      owner.Name,
		OpsrcOwnerNamespaceLabel: owner.Namespace,
	}

	for key, value := range b.object.GetLabels() {
		labels[key] = value
	}

	b.object.SetLabels(labels)
	return b
}

// WithSpec sets Spec accordingly.
func (b *CatalogSourceConfigBuilder) WithSpec(targetNamespace, packages, displayName, publisher string) *CatalogSourceConfigBuilder {
	b.object.Spec = marketplace.CatalogSourceConfigSpec{
		TargetNamespace: targetNamespace,
		Packages:        packages,
		DisplayName:     displayName,
		Publisher:       publisher,
	}

	return b
}
