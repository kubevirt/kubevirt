package imageupload

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	instancetypeapi "kubevirt.io/api/instancetype"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/apimachinery/pkg/fields"

	"kubevirt.io/kubevirt/pkg/pointer"
)

func (c *command) parseArgs(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expecting two args")
	}

	switch strings.ToLower(args[0]) {
	case "dv":
		c.createPVC = false
	case "pvc":
		c.createPVC = true
	default:
		return fmt.Errorf("invalid resource type %s", args[0])
	}

	c.name = args[1]

	return nil
}

func (c *command) createStorageSpec() (*cdiv1.StorageSpec, error) {
	quantity, err := resource.ParseQuantity(c.size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", c.size, err)
	}

	spec := &cdiv1.StorageSpec{
		Resources: v1.VolumeResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: quantity,
			},
		},
	}

	if c.storageClass != "" {
		spec.StorageClassName = &c.storageClass
	}

	if c.accessMode != "" {
		if c.accessMode == string(v1.ReadOnlyMany) {
			return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
		}
		spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(c.accessMode)}
	}

	switch c.volumeMode {
	case "block":
		spec.VolumeMode = pointer.P(v1.PersistentVolumeBlock)
	case "filesystem":
		spec.VolumeMode = pointer.P(v1.PersistentVolumeFilesystem)
	}

	return spec, nil
}

func (c *command) setDefaultInstancetypeLabels(target metav1.Object) {
	labels := target.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		target.SetLabels(labels)
	}

	if c.defaultInstancetype != "" {
		labels[instancetypeapi.DefaultInstancetypeLabel] = c.defaultInstancetype
		// Kind is optional, defaults to cluster-wide setting
		if c.defaultInstancetypeKind != "" {
			labels[instancetypeapi.DefaultInstancetypeKindLabel] = c.defaultInstancetypeKind
		}
	}
	if c.defaultPreference != "" {
		labels[instancetypeapi.DefaultPreferenceLabel] = c.defaultPreference
		// Kind is optional, defaults to cluster-wide setting
		if c.defaultPreferenceKind != "" {
			labels[instancetypeapi.DefaultPreferenceKindLabel] = c.defaultPreferenceKind
		}
	}
}

// handleEventErrors checks PVC and DV-related events and, when encountered, returns appropriate errors
func (c *command) handleEventErrors(pvcName, dvName string) error {
	var pvcUID types.UID
	var dvUID types.UID

	if pvcName != "" {
		pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if pvc != nil {
			pvcUID = pvc.GetUID()
		}
	}

	if dvName != "" {
		dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Get(context.Background(), dvName, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if dv != nil {
			dvUID = dv.GetUID()
		}
	}

	// Retrieve events filtered by involved object
	eventList, err := c.client.CoreV1().Events(c.namespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("involvedObject.name", c.name).String(),
	})
	if err != nil {
		return err
	}

	// TODO: Currently, we only check 'provisioningFailed' and 'errClaimNotValid' events.
	// If necessary, support more relevant errors
	if pvcUID == "" && dvUID == "" {
		// No relevant events to process
		return nil
	}

	for _, event := range eventList.Items {
		objectKind := event.InvolvedObject.Kind
		objectUID := event.InvolvedObject.UID

		if objectUID == pvcUID || objectUID == dvUID {
			if objectKind == "PersistentVolumeClaim" && event.Reason == provisioningFailed &&
				!strings.Contains(event.Message, optimisticLockErrorMsg) {
				return fmt.Errorf("Provisioning failed: %s", event.Message)
			}

			if objectKind == "DataVolume" && event.Reason == errClaimNotValid {
				return fmt.Errorf("Claim not valid: %s", event.Message)
			}
		}
	}

	return nil
}
