package operatorstatus

import (
	"errors"
	"reflect"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// NewWriter returns a new instance of Writer.
func NewWriter(discovery discovery.DiscoveryInterface, client configv1client.ConfigV1Interface) *Writer {
	return &Writer{
		discovery: discovery,
		client:    client,
	}
}

// Writer encapsulates logic for cluster operator object API. It is used to
// update ClusterOperator resource.
type Writer struct {
	discovery discovery.DiscoveryInterface
	client    configv1client.ConfigV1Interface
}

// EnsureExists ensures that the cluster operator resource exists with a default
// status that reflects expecting status.
func (w *Writer) EnsureExists(name string) (existing *configv1.ClusterOperator, err error) {
	existing, err = w.client.ClusterOperators().Get(name, metav1.GetOptions{})
	if err == nil {
		return
	}

	if !k8serrors.IsNotFound(err) {
		return
	}

	co := &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	existing, err = w.client.ClusterOperators().Create(co)
	return
}

// UpdateStatus updates the clusteroperator object with the new status specified.
func (w *Writer) UpdateStatus(existing *configv1.ClusterOperator, newStatus *configv1.ClusterOperatorStatus) error {
	if newStatus == nil || existing == nil {
		return errors.New("input specified is <nil>")
	}

	existingStatus := existing.Status.DeepCopy()
	if reflect.DeepEqual(existingStatus, newStatus) {
		return nil
	}

	existing.Status = *newStatus
	if _, err := w.client.ClusterOperators().UpdateStatus(existing); err != nil {
		return err
	}

	return nil
}

// IsAPIAvailable return true if cluster operator API is present on the cluster.
// Otherwise, exists is set to false.
func (w *Writer) IsAPIAvailable() (exists bool, err error) {
	opStatusGV := schema.GroupVersion{
		Group:   "config.openshift.io",
		Version: "v1",
	}
	err = discovery.ServerSupportsVersion(w.discovery, opStatusGV)
	if err != nil {
		return
	}

	exists = true
	return
}
