package components

import (
	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	virtv1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
)

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
				"prometheus.kubevirt.io": "",
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

func NewOpertorValidatingWebhookConfiguration(operatorNamespace string) *v1beta1.ValidatingWebhookConfiguration {
	failurePolicy := v1beta1.Fail
	sideEffectNone := v1beta1.SideEffectClassNone
	path := "/kubevirt-validate-delete"

	return &v1beta1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: KubeVirtOperatorValidatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel: KubeVirtOperatorValidatingWebhookName,
			},
			Annotations: map[string]string{
				"certificates.kubevirt.io/secret": "kubevirt-operator-certs",
			},
		},
		Webhooks: []v1beta1.ValidatingWebhook{
			{
				Name: "kubevirt-validator.kubevirt.io",
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: operatorNamespace,
						Name:      VirtOperatorServiceName,
						Path:      &path,
					},
				},
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Delete,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"kubevirts"},
					},
				}},
				FailurePolicy: &failurePolicy,
				SideEffects:   &sideEffectNone,
			},
		},
	}
}

func NewVirtAPIMutatingWebhookConfiguration(installNamespace string) *v1beta1.MutatingWebhookConfiguration {
	vmPath := VMMutatePath
	vmiPath := VMIMutatePath
	migrationPath := MigrationMutatePath

	return &v1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VirtAPIMutatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel:       VirtAPIMutatingWebhookName,
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
			Annotations: map[string]string{
				"certificates.kubevirt.io/secret": VirtApiCertSecretName,
			},
		},
		Webhooks: []v1beta1.MutatingWebhook{
			{
				Name: "virtualmachines-mutator.kubevirt.io",
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachines"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmPath,
					},
				},
			},
			{
				Name: "virtualmachineinstances-mutator.kubevirt.io",
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPath,
					},
				},
			},
			{
				Name: "migrations-mutator.kubevirt.io",
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationPath,
					},
				},
			},
		},
	}

}

func NewVirtAPIValidatingWebhookConfiguration(installNamespace string) *v1beta1.ValidatingWebhookConfiguration {

	vmiPathCreate := VMICreateValidatePath
	vmiPathUpdate := VMIUpdateValidatePath
	vmPath := VMValidatePath
	vmirsPath := VMIRSValidatePath
	vmipresetPath := VMIPresetValidatePath
	migrationCreatePath := MigrationCreateValidatePath
	migrationUpdatePath := MigrationUpdateValidatePath
	vmSnapshotValidatePath := VMSnapshotValidatePath
	failurePolicy := v1beta1.Fail

	return &v1beta1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VirtAPIValidatingWebhookName,
			Labels: map[string]string{
				virtv1.AppLabel:       VirtAPIValidatingWebhookName,
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
			Annotations: map[string]string{
				"certificates.kubevirt.io/secret": VirtApiCertSecretName,
			},
		},
		Webhooks: []v1beta1.ValidatingWebhook{
			{
				Name:          "virtualmachineinstances-create-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPathCreate,
					},
				},
			},
			{
				Name:          "virtualmachineinstances-update-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstances"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmiPathUpdate,
					},
				},
			},
			{
				Name:          "virtualmachine-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachines"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmPath,
					},
				},
			},
			{
				Name:          "virtualmachinereplicaset-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancereplicasets"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmirsPath,
					},
				},
			},
			{
				Name:          "virtualmachinepreset-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancepresets"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmipresetPath,
					},
				},
			},
			{
				Name:          "migration-create-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationCreatePath,
					},
				},
			},
			{
				Name:          "migration-update-validator.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{virtv1.GroupName},
						APIVersions: virtv1.ApiSupportedWebhookVersions,
						Resources:   []string{"virtualmachineinstancemigrations"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &migrationUpdatePath,
					},
				},
			},
			{
				Name:          "virtualmachinesnapshot-validator.snapshot.kubevirt.io",
				FailurePolicy: &failurePolicy,
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{
						v1beta1.Create,
						v1beta1.Update,
					},
					Rule: v1beta1.Rule{
						APIGroups:   []string{snapshotv1.SchemeGroupVersion.Group},
						APIVersions: []string{snapshotv1.SchemeGroupVersion.Version},
						Resources:   []string{"virtualmachinesnapshots"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: installNamespace,
						Name:      VirtApiServiceName,
						Path:      &vmSnapshotValidatePath,
					},
				},
			},
		},
	}
}

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

const VirtHandlerServiceName = "virt-controller"

const VirtAPIValidatingWebhookName = "virt-api-validator"

const VirtOperatorServiceName = "kubevirt-operator-webhook"

const VirtAPIMutatingWebhookName = "virt-api-mutator"

const KubevirtOperatorWebhookServiceName = "kubevirt-operator-webhook"

const KubeVirtOperatorValidatingWebhookName = "virt-operator-validator"

const VMSnapshotValidatePath = "/virtualmachinesnapshots-validate"
