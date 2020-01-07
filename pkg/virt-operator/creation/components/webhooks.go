package components

import (
	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	virtv1 "kubevirt.io/client-go/api/v1"
)

func NewWebhookService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-operator-webhook",
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

func NewValidatingWebhookConfiguration(namespace string) *v1beta1.ValidatingWebhookConfiguration {
	failurePolicy := v1beta1.Fail
	sideEffectNone := v1beta1.SideEffectClassNone
	path := "/kubevirt-validate-delete"

	return &v1beta1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-operator-validator",
			Labels: map[string]string{
				virtv1.AppLabel: "virt-operator-validator",
			},
		},
		Webhooks: []v1beta1.Webhook{
			{
				Name: "kubevirt-validator.kubevirt.io",
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: namespace,
						Name:      "kubevirt-operator-webhook",
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
