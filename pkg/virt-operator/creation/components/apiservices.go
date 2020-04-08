package components

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	v1 "kubevirt.io/client-go/api/v1"
)

func NewVirtAPIAPIServices(installNamespace string) []*v1beta1.APIService {
	apiservices := []*v1beta1.APIService{}

	for _, version := range v1.SubresourceGroupVersions {
		subresourceAggregatedApiName := version.Version + "." + version.Group

		apiservices = append(apiservices, &v1beta1.APIService{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiregistration.k8s.io/v1beta1",
				Kind:       "APIService",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: subresourceAggregatedApiName,
				Labels: map[string]string{
					v1.AppLabel:       "virt-api-aggregator",
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
				Annotations: map[string]string{
					"certificates.kubevirt.io/secret": VirtApiCertSecretName,
				},
			},
			Spec: v1beta1.APIServiceSpec{
				Service: &v1beta1.ServiceReference{
					Namespace: installNamespace,
					Name:      VirtApiServiceName,
				},
				Group:                version.Group,
				Version:              version.Version,
				GroupPriorityMinimum: 1000,
				VersionPriority:      15,
			},
		})
	}
	return apiservices
}
