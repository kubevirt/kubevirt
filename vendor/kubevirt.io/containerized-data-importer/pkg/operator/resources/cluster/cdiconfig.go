package cluster

import (
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createCDIConfigCRD() *extv1beta1.CustomResourceDefinition {
	return &extv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cdiconfigs.cdi.kubevirt.io",
			Labels: map[string]string{
				"cdi.kubevirt.io": "",
			},
		},
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Group: "cdi.kubevirt.io",
			Names: extv1beta1.CustomResourceDefinitionNames{
				Kind:     "CDIConfig",
				Plural:   "cdiconfigs",
				Singular: "cdiconfig",
			},
			Version: "v1alpha1",
			Scope:   "Cluster",
		},
	}
}
