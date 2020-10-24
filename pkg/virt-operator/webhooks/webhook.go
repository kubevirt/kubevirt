package webhooks

import (
	"encoding/json"
	"fmt"
	"math"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

const (
	uninstallErrorMsg            = "Rejecting the uninstall request, since there are still %s present. Either delete all KubeVirt related workloads or change the uninstall strategy before uninstalling KubeVirt."
	repeatingBucketsErrorMsg     = "Make sure bucket values don't repeat."
	unorderedBucketsErrorMsg     = "Make sure bucket values are properly ordered."
	invalidInitialBucketErrorMsg = "Initial bucket value must be greater than 1."
	insufficientBucketsErrorMsg  = "BucketValues field must have 2 or more elements."
	missingFieldErrorMsg         = "Missing bucketValues field for %v metrics."
)

var KubeVirtGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceGroupVersionKind.Version,
	Resource: "kubevirts",
}

func NewKubeVirtDeletionAdmitter(client kubecli.KubevirtClient) *KubeVirtDeletionAdmitter {
	return &KubeVirtDeletionAdmitter{
		client: client,
	}
}

type KubeVirtDeletionAdmitter struct {
	client kubecli.KubevirtClient
}

func (k *KubeVirtDeletionAdmitter) Admit(review *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	var kv *v1.KubeVirt
	var err error
	if review.Request.Name != "" {
		kv, err = k.client.KubeVirt(review.Request.Namespace).Get(review.Request.Name, &metav1.GetOptions{})
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
	} else {
		list, err := k.client.KubeVirt(review.Request.Namespace).List(&metav1.ListOptions{})
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		if len(list.Items) == 0 {
			return validating_webhooks.NewPassingAdmissionResponse()
		} else {
			kv = &list.Items[0]
		}
	}

	if kv.Spec.UninstallStrategy == "" || kv.Spec.UninstallStrategy == v1.KubeVirtUninstallStrategyRemoveWorkloads {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	if kv.Status.Phase != v1.KubeVirtPhaseDeployed {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmis, err := k.client.VirtualMachineInstance(metav1.NamespaceAll).List(&metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vmis.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machine Instances"))
	}

	vms, err := k.client.VirtualMachine(metav1.NamespaceAll).List(&metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vms.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machines"))
	}

	vmirs, err := k.client.ReplicaSet(metav1.NamespaceAll).List(metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vmirs.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machine Instance Replica Sets"))
	}

	return validating_webhooks.NewPassingAdmissionResponse()
}

func NewKubeVirtMutationAdmitter(client kubecli.KubevirtClient) *KubeVirtMutationAdmitter {
	return &KubeVirtMutationAdmitter{
		client: client,
	}
}

type KubeVirtMutationAdmitter struct {
	client kubecli.KubevirtClient
}

func (k *KubeVirtMutationAdmitter) Admit(review *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	raw := review.Request.Object.Raw
	kv := v1.KubeVirt{}

	err := json.Unmarshal(raw, &kv)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// No configuration was provided, default values will be used
	if kv.Spec.MetricsConfig == nil {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	return validateMetricsConfig(kv.Spec.MetricsConfig)
}

func validateMetricsConfig(metricsConfig *v1.MetricsConfig) *v1beta1.AdmissionResponse {
	if metricsConfig.MigrationMetrics != nil {
		err := validateDurationBuckets(metricsConfig.MigrationMetrics.DurationHistogram, "migration")
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
	}
	// Add more validations here when developing new histograms

	// Everything is valid
	// Use default values for missing configuration
	return validating_webhooks.NewPassingAdmissionResponse()
}

// validateDurationBuckets validates histogram buckets of time related metrics
func validateDurationBuckets(histogram *v1.HistogramMetric, metricType string) error {
	if histogram == nil || histogram.BucketValues == nil {
		return fmt.Errorf(missingFieldErrorMsg, metricType)
	}

	lastBucket := float64(math.MinInt64)
	for _, bucket := range histogram.BucketValues {
		if bucket < lastBucket {
			return fmt.Errorf(unorderedBucketsErrorMsg)
		}
		if bucket == lastBucket {
			return fmt.Errorf(repeatingBucketsErrorMsg)
		}
		lastBucket = bucket
	}

	if histogram.BucketValues[0] < 1 {
		return fmt.Errorf(invalidInitialBucketErrorMsg)
	}

	if len(histogram.BucketValues) < 2 {
		return fmt.Errorf(insufficientBucketsErrorMsg)
	}

	// If it got here, it means the configuration is valid
	return nil
}
