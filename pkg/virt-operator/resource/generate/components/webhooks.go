package components

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/api/core"

	virtv1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
)

var sideEffectNone = admissionregistrationv1.SideEffectClassNone
var sideEffectNoneOnDryRun = admissionregistrationv1.SideEffectClassNoneOnDryRun

const certificatesSecretAnnotationKey = "certificates.kubevirt.io/secret"

var defaultTimeoutSeconds = int32(10)

func NewOperatorWebhookService(operatorNamespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      KubevirtOperatorWebhookServiceName,
			Labels: map[string]string{
				virtv1.AppLabel:          "",
				"prometheus.kubevirt.io": prometheusLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"kubevirt.io": "virt-operator",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "webhooks",
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "webhooks",
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func NewOpertorValidatingWebhookConfiguration(operatorNamespace string) *admissionregistrationv1.ValidatingWebhookConfiguration {
	failurePolicy := admissionregistrationv1.Fail
	path := "/kubevirt-validate-delete"
	kubevirtUpdatePath := KubeVirtUpdateValidatePath

	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: KubeVirtOperatorValidatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel: KubeVirtOperatorValidatingWebhookName,
			},
			Annotations: map[string]string{
				certificatesSecretAnnotationKey: "kubevirt-operator-certs",
			},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "kubevirt-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: operatorNamespace,
						Name:      VirtOperatorServiceName,
						Path:      &path,
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Delete,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"kubevirts"},
					},
				}},
				FailurePolicy:  &failurePolicy,
				TimeoutSeconds: &defaultTimeoutSeconds,
				SideEffects:    &sideEffectNone,
			},
			{
				Name:                    "kubevirt-update-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"kubevirts"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: operatorNamespace,
						Name:      VirtOperatorServiceName,
						Path:      &kubevirtUpdatePath,
					},
				},
			},
		},
	}
}

func NewVirtAPIMutatingWebhookConfiguration(installNamespace string) *admissionregistrationv1.MutatingWebhookConfiguration {
	vmPath := VMMutatePath
	vmiPath := VMIMutatePath
	migrationPath := MigrationMutatePath
	failurePolicy := admissionregistrationv1.Fail

	return &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VirtAPIMutatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel:       VirtAPIMutatingWebhookName,
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
			Annotations: map[string]string{
				certificatesSecretAnnotationKey: VirtApiCertSecretName,
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name:                    "virtualmachines-mutator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachines"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmPath,
					},
				},
			},
			{
				Name:                    "virtualmachineinstances-mutator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPath,
					},
				},
			},
			{
				Name:                    "migrations-mutator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationPath,
					},
				},
			},
		},
	}

}

func NewVirtAPIValidatingWebhookConfiguration(installNamespace string) *admissionregistrationv1.ValidatingWebhookConfiguration {
	vmiPathCreate := VMICreateValidatePath
	vmiPathUpdate := VMIUpdateValidatePath
	vmPath := VMValidatePath
	vmirsPath := VMIRSValidatePath
	vmipresetPath := VMIPresetValidatePath
	migrationCreatePath := MigrationCreateValidatePath
	migrationUpdatePath := MigrationUpdateValidatePath
	vmSnapshotValidatePath := VMSnapshotValidatePath
	vmRestoreValidatePath := VMRestoreValidatePath
	VmFlavorValidatePath := VMFlavorValidatePath
	VmClusterFlavorValidatePath := VMClusterFlavorValidatePath
	launcherEvictionValidatePath := LauncherEvictionValidatePath
	statusValidatePath := StatusValidatePath
	failurePolicy := admissionregistrationv1.Fail
	ignorePolicy := admissionregistrationv1.Ignore

	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VirtAPIValidatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel:       VirtAPIValidatingWebhookName,
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
			Annotations: map[string]string{
				certificatesSecretAnnotationKey: VirtApiCertSecretName,
			},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "virt-launcher-eviction-interceptor.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				// We don't want to block evictions in the cluster in a case where this webhook is down.
				// The eviction of virt-launcher will still be protected by our pdb.
				FailurePolicy:  &ignorePolicy,
				TimeoutSeconds: &defaultTimeoutSeconds,
				SideEffects:    &sideEffectNoneOnDryRun,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.OperationAll,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods/eviction"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &launcherEvictionValidatePath,
					},
				},
			},
			{
				Name:                    "virtualmachineinstances-create-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPathCreate,
					},
				},
			},
			{
				Name:                    "virtualmachineinstances-update-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPathUpdate,
					},
				},
			},
			{
				Name:                    "virtualmachine-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachines"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmPath,
					},
				},
			},
			{
				Name:                    "virtualmachinereplicaset-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancereplicasets"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmirsPath,
					},
				},
			},
			{
				Name:                    "virtualmachinepreset-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancepresets"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmipresetPath,
					},
				},
			},
			{
				Name:                    "migration-create-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationCreatePath,
					},
				},
			},
			{
				Name:                    "migration-update-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationUpdatePath,
					},
				},
			},
			{
				Name:                    "virtualmachinesnapshot-validator.snapshot.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{snapshotv1.SchemeGroupVersion.Group},
						APIVersions: []string{snapshotv1.SchemeGroupVersion.Version},
						Resources:   []string{"virtualmachinesnapshots"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmSnapshotValidatePath,
					},
				},
			},
			{
				Name:                    "virtualmachinerestore-validator.snapshot.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{snapshotv1.SchemeGroupVersion.Group},
						APIVersions: []string{snapshotv1.SchemeGroupVersion.Version},
						Resources:   []string{"virtualmachinerestores"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmRestoreValidatePath,
					},
				},
			},
			{
				Name:                    "virtualmachineflavor-validator.flavor.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{flavorv1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{flavorv1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"virtualmachineflavors"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &VmFlavorValidatePath,
					},
				},
			},
			{
				Name:                    "virtualmachineclusterflavor-validator.flavor.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{flavorv1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{flavorv1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"virtualmachineclusterflavors"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &VmClusterFlavorValidatePath,
					},
				},
			},
			{
				Name:                    "kubevirt-crd-status-validator.kubevirt.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &defaultTimeoutSeconds,
				SideEffects:             &sideEffectNone,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{core.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources: []string{
							"virtualmachines/status",
							"virtualmachineinstancereplicasets/status",
							"virtualmachineinstancemigrations/status",
						},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &statusValidatePath,
					},
				},
			},
		},
	}
}

const KubeVirtUpdateValidatePath = "/kubevirt-validate-update"

const VMICreateValidatePath = "/virtualmachineinstances-validate-create"

const VMIUpdateValidatePath = "/virtualmachineinstances-validate-update"

const VMValidatePath = "/virtualmachines-validate"

const VMIRSValidatePath = "/virtualmachinereplicaset-validate"

const VMIPresetValidatePath = "/vmipreset-validate"

const MigrationCreateValidatePath = "/migration-validate-create"

const MigrationUpdateValidatePath = "/migration-validate-update"

const VMMutatePath = "/virtualmachines-mutate"

const VMIMutatePath = "/virtualmachineinstances-mutate"

const MigrationMutatePath = "/migration-mutate-create"

const VirtApiServiceName = "virt-api"

const VirtControllerServiceName = "virt-controller"

const VirtHandlerServiceName = "virt-handler"

const VirtAPIValidatingWebhookName = "virt-api-validator"

const VirtOperatorServiceName = "kubevirt-operator-webhook"

const VirtAPIMutatingWebhookName = "virt-api-mutator"

const KubevirtOperatorWebhookServiceName = "kubevirt-operator-webhook"

const KubeVirtOperatorValidatingWebhookName = "virt-operator-validator"

const VMSnapshotValidatePath = "/virtualmachinesnapshots-validate"

const VMRestoreValidatePath = "/virtualmachinerestores-validate"

const VMFlavorValidatePath = "/virtualmachineflavors-validate"

const VMClusterFlavorValidatePath = "/virtualmachineclusterflavors-validate"

const StatusValidatePath = "/status-validate"

const LauncherEvictionValidatePath = "/launcher-eviction-validate"
