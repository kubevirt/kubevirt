/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package apply

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	k6tv1 "kubevirt.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"

	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	policyv1 "k8s.io/api/policy/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func getGroupResource(required runtime.Object) (group string, resource string, err error) {

	switch required.(type) {
	case *extv1.CustomResourceDefinition:
		group = "apiextensions.k8s.io/v1"
		resource = "customresourcedefinitions"
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "mutatingwebhookconfigurations"
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "validatingwebhookconfigurations"
	case *policyv1.PodDisruptionBudget:
		group = "apps"
		resource = "poddisruptionbudgets"
	case *appsv1.Deployment:
		group = "apps"
		resource = "deployments"
	case *appsv1.DaemonSet:
		group = "apps"
		resource = "daemonsets"
	default:
		err = fmt.Errorf("resource type is not known")
		return
	}

	return
}

func GetExpectedGeneration(required runtime.Object, previousGenerations []k6tv1.GenerationStatus) int64 {
	group, resource, err := getGroupResource(required)
	if err != nil {
		return -1
	}

	operatorGenerations := toOperatorGenerations(previousGenerations)

	meta := required.(v1.Object)
	generation := resourcemerge.GenerationFor(operatorGenerations, schema.GroupResource{Group: group, Resource: resource}, meta.GetNamespace(), meta.GetName())
	if generation == nil {
		return -1
	}

	return generation.LastGeneration
}

func SetGeneration(generations *[]k6tv1.GenerationStatus, actual runtime.Object) {

	if actual == nil {
		return
	}

	group, resource, err := getGroupResource(actual)
	if err != nil {
		return
	}

	operatorGenerations := toOperatorGenerations(*generations)
	meta := actual.(v1.Object)

	resourcemerge.SetGeneration(&operatorGenerations, operatorsv1.GenerationStatus{
		Group:          group,
		Resource:       resource,
		Namespace:      meta.GetNamespace(),
		Name:           meta.GetName(),
		LastGeneration: meta.GetGeneration(),
	})

	newGenerations := toAPIGenerations(operatorGenerations)
	*generations = newGenerations
}

func toOperatorGeneration(generation k6tv1.GenerationStatus) operatorsv1.GenerationStatus {
	return operatorsv1.GenerationStatus{
		Group:          generation.Group,
		Resource:       generation.Resource,
		Namespace:      generation.Namespace,
		Name:           generation.Name,
		LastGeneration: generation.LastGeneration,
		Hash:           generation.Hash,
	}
}

func toAPIGeneration(generation operatorsv1.GenerationStatus) k6tv1.GenerationStatus {
	return k6tv1.GenerationStatus{
		Group:          generation.Group,
		Resource:       generation.Resource,
		Namespace:      generation.Namespace,
		Name:           generation.Name,
		LastGeneration: generation.LastGeneration,
		Hash:           generation.Hash,
	}
}

func toOperatorGenerations(generations []k6tv1.GenerationStatus) (operatorGenerations []operatorsv1.GenerationStatus) {
	for _, generation := range generations {
		operatorGenerations = append(operatorGenerations, toOperatorGeneration(generation))
	}
	return operatorGenerations
}

func toAPIGenerations(generations []operatorsv1.GenerationStatus) (apiGenerations []k6tv1.GenerationStatus) {
	for _, generation := range generations {
		apiGenerations = append(apiGenerations, toAPIGeneration(generation))
	}
	return apiGenerations
}
