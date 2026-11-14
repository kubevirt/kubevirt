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
 *
 */

package webhooks

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var _ = Describe("webhook source guard", func() {
	tenantPod := func(ip string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant", Namespace: "default"},
			Status: corev1.PodStatus{
				PodIP:  ip,
				PodIPs: []corev1.PodIP{{IP: ip}, {IP: "fd00::10"}},
			},
		}
	}

	newGuard := func(synced bool, pods ...*corev1.Pod) *PodSourceGuard {
		return &PodSourceGuard{
			hasSynced: func() bool { return synced },
			listPods:  func() []*corev1.Pod { return pods },
		}
	}

	It("allows every caller until the pod cache has synced", func() {
		guard := newGuard(false, tenantPod("10.1.2.3"))
		Expect(guard.Allows("10.1.2.3:40000")).To(BeTrue())
	})

	It("rejects a tenant pod IP", func() {
		guard := newGuard(true, tenantPod("10.1.2.3"))
		Expect(guard.Allows("10.1.2.3:40000")).To(BeFalse())
		Expect(guard.Allows("[fd00::10]:40000")).To(BeFalse())
	})

	It("rejects an IPv4-mapped form of a tenant pod IP", func() {
		guard := newGuard(true, tenantPod("10.1.2.3"))
		Expect(guard.Allows("[::ffff:10.1.2.3]:40000")).To(BeFalse())
	})

	It("allows a kube-system pod IP", func() {
		guard := newGuard(true, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "konnectivity-agent", Namespace: metav1.NamespaceSystem},
			Status:     corev1.PodStatus{PodIP: "10.1.2.3"},
		})
		Expect(guard.Allows("10.1.2.3:40000")).To(BeTrue())
	})

	It("allows a hostNetwork pod IP", func() {
		guard := newGuard(true, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver", Namespace: "openshift-kube-apiserver"},
			Spec:       corev1.PodSpec{HostNetwork: true},
			Status:     corev1.PodStatus{PodIP: "192.168.1.10"},
		})
		Expect(guard.Allows("192.168.1.10:443")).To(BeTrue())
	})

	It("allows an address that is not a pod IP", func() {
		guard := newGuard(true, tenantPod("10.1.2.3"))
		Expect(guard.Allows("192.168.1.10:443")).To(BeTrue())
	})

	It("uses PodIP when PodIPs is empty", func() {
		guard := newGuard(true, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant", Namespace: "default"},
			Status:     corev1.PodStatus{PodIP: "10.9.8.7"},
		})
		Expect(guard.Allows("10.9.8.7:1234")).To(BeFalse())
	})

	It("does not call the webhook handler when the source is rejected", func() {
		guard := newGuard(true, tenantPod("10.1.2.3"))
		called := false
		handler := guard.Handler(func(http.ResponseWriter, *http.Request) {
			called = true
		})

		req := httptest.NewRequest(http.MethodPost, "/kubevirt-validate-update", nil)
		req.RemoteAddr = "10.1.2.3:40000"
		recorder := httptest.NewRecorder()
		handler(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
		Expect(called).To(BeFalse())
	})

	It("calls the webhook handler for a non-pod source", func() {
		guard := newGuard(true, tenantPod("10.1.2.3"))
		called := false
		handler := guard.Handler(func(http.ResponseWriter, *http.Request) {
			called = true
		})

		req := httptest.NewRequest(http.MethodPost, "/kubevirt-validate-update", nil)
		req.RemoteAddr = "192.168.1.10:443"
		recorder := httptest.NewRecorder()
		handler(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(called).To(BeTrue())
	})

	It("keeps only the fields the source guard reads", func() {
		full := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tenant",
				Namespace: "default",
				Labels:    map[string]string{"app": "keep-me-out"},
			},
			Spec: corev1.PodSpec{
				HostNetwork: false,
				Containers:  []corev1.Container{{Name: "app", Image: "example"}},
			},
			Status: corev1.PodStatus{
				PodIP:  "10.1.2.3",
				PodIPs: []corev1.PodIP{{IP: "10.1.2.3"}},
				Phase:  corev1.PodRunning,
			},
		}

		trimmed, err := trimWebhookSourcePod(full)
		Expect(err).NotTo(HaveOccurred())
		pod := trimmed.(*corev1.Pod)
		Expect(pod.Name).To(Equal("tenant"))
		Expect(pod.Namespace).To(Equal("default"))
		Expect(pod.Labels).To(BeEmpty())
		Expect(pod.Spec.Containers).To(BeEmpty())
		Expect(pod.Spec.HostNetwork).To(BeFalse())
		Expect(pod.Status.PodIP).To(Equal("10.1.2.3"))
		Expect(pod.Status.Phase).To(BeEmpty())

		wrapped, err := trimWebhookSourcePod(cache.DeletedFinalStateUnknown{Obj: full})
		Expect(err).NotTo(HaveOccurred())
		Expect(wrapped.(*corev1.Pod).Status.PodIP).To(Equal("10.1.2.3"))
	})
})
