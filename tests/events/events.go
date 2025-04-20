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

package events

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"kubevirt.io/client-go/kubecli"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type k8sObject interface {
	metav1.Object
	schema.ObjectKind
}

// ExpectNoEvent is safe to use in parallel as long as you are asserting namespaced object that is not shared between tests
func ExpectNoEvent(object k8sObject, eventType, reason string) {
	By("Expecting for an event to be not triggered")
	expectEvent(object, eventType, reason, BeEmpty())
}

// ExpectEvent is safe to use in parallel as long as you are asserting namespaced object that is not shared between tests
func ExpectEvent(object k8sObject, eventType, reason string) {
	By("Expecting for an event to be triggered")
	expectEvent(object, eventType, reason, Not(BeEmpty()))
}

// DeleteEvents is safe to use in parallel as long as you are asserting namespaced object that is not shared between tests
func DeleteEvents(object k8sObject, eventType, reason string) {
	By("Expecting events to be removed")
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

	fieldSelector, namespace := constructFieldSelectorAndNamespace(object, eventType, reason)

	events, err := virtClient.CoreV1().Events(namespace).List(context.Background(),
		metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	for _, event := range events.Items {
		err = virtClient.CoreV1().Events(event.Namespace).Delete(
			context.TODO(),
			event.Name,
			metav1.DeleteOptions{},
		)
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("failed to delete event %s/%s", event.Namespace, event.Name))
	}

	EventuallyWithOffset(1, func() []k8sv1.Event {
		events, err := virtClient.CoreV1().Events(namespace).List(context.Background(),
			metav1.ListOptions{
				FieldSelector: fieldSelector,
			})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		return events.Items
	}, 30*time.Second, 1*time.Second).Should(BeEmpty(), fmt.Sprintf("Used fieldselector %s", fieldSelector))
}

func expectEvent(object k8sObject, eventType, reason string, matcher types.GomegaMatcher) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(2, err).ToNot(HaveOccurred())

	fieldSelector, namespace := constructFieldSelectorAndNamespace(object, eventType, reason)

	EventuallyWithOffset(2, func() []k8sv1.Event {
		events, err := virtClient.CoreV1().Events(namespace).List(
			context.Background(),
			metav1.ListOptions{
				FieldSelector: fieldSelector,
			},
		)
		ExpectWithOffset(3, err).ToNot(HaveOccurred())
		return events.Items
	}, 30*time.Second).Should(matcher, fmt.Sprintf("Used fieldselector %s", fieldSelector))
}

// constructFieldSelectorAndNamespace does best effort to overcome https://github.com/kubernetes/client-go/issues/861
func constructFieldSelectorAndNamespace(object k8sObject, eventType, reason string) (string, string) {
	kind := object.GroupVersionKind().Kind
	if kind == "" {
		kind = reflect.ValueOf(object).Type().Name()
	}
	kindSelector := fmt.Sprintf("involvedObject.kind=%s,", kind)
	if kind == "" {
		kindSelector = ""
	}

	name := object.GetName()
	namespace := object.GetNamespace()

	fieldSelector := fmt.Sprintf("%sinvolvedObject.name=%s,type=%s,reason=%s", kindSelector, name, eventType, reason)
	return fieldSelector, namespace
}
