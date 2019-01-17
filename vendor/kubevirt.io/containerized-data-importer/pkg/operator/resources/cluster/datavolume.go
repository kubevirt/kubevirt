/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createCRDResources(args *FactoryArgs) []Resource {
	return []Resource{
		createDataVolumeCRD(),
	}
}

func createDataVolumeCRD() *extv1beta1.CustomResourceDefinition {
	return &extv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "datavolumes.cdi.kubevirt.io",
			Labels: map[string]string{
				"cdi.kubevirt.io": "",
			},
		},
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Group: "cdi.kubevirt.io",
			Names: extv1beta1.CustomResourceDefinitionNames{
				Kind:   "DataVolume",
				Plural: "datavolumes",
				ShortNames: []string{
					"dv",
					"dvs",
				},
				Singular: "datavolume",
			},
			Version: "v1alpha1",
			Scope:   "Namespaced",
			Validation: &extv1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &extv1beta1.JSONSchemaProps{
					Properties: map[string]extv1beta1.JSONSchemaProps{
						"apiVersion": extv1beta1.JSONSchemaProps{
							Type: "string",
						},
						"kind": extv1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": extv1beta1.JSONSchemaProps{},
						"spec": extv1beta1.JSONSchemaProps{
							Properties: map[string]extv1beta1.JSONSchemaProps{
								"source": extv1beta1.JSONSchemaProps{
									Properties: map[string]extv1beta1.JSONSchemaProps{
										"http": extv1beta1.JSONSchemaProps{
											Properties: map[string]extv1beta1.JSONSchemaProps{
												"url": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
												"secretRef": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
											Required: []string{
												"url",
											},
										},
										"s3": extv1beta1.JSONSchemaProps{
											Properties: map[string]extv1beta1.JSONSchemaProps{
												"url": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
												"secretRef": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
											Required: []string{
												"url",
											},
										},
										"registry": extv1beta1.JSONSchemaProps{
											Properties: map[string]extv1beta1.JSONSchemaProps{
												"url": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
												"secretRef": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
											Required: []string{
												"url",
											},
										},
										"pvc": extv1beta1.JSONSchemaProps{
											Properties: map[string]extv1beta1.JSONSchemaProps{
												"namespace": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
												"name": extv1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
											Required: []string{
												"namespace",
												"name",
											},
										},
										"upload": extv1beta1.JSONSchemaProps{},
										"blank":  extv1beta1.JSONSchemaProps{},
									},
								},
								"pvc": extv1beta1.JSONSchemaProps{
									Properties: map[string]extv1beta1.JSONSchemaProps{
										"resources": extv1beta1.JSONSchemaProps{
											Properties: map[string]extv1beta1.JSONSchemaProps{
												"requests": extv1beta1.JSONSchemaProps{
													Properties: map[string]extv1beta1.JSONSchemaProps{
														"storage": extv1beta1.JSONSchemaProps{
															Type: "string",
														},
													},
												},
											},
											Required: []string{
												"requests",
											},
										},
										"storageClassName": extv1beta1.JSONSchemaProps{
											Type: "string",
										},
										"accessModes": extv1beta1.JSONSchemaProps{
											Type: "array",
										},
									},
									Required: []string{
										"resources",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
