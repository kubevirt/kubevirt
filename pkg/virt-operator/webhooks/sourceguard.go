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
	"net"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
)

// PodSourceGuard rejects virt-operator admission webhook calls that come from
// an ordinary pod.
//
// kube-apiserver does not present a client certificate to admission webhooks
// unless the cluster administrator configures --admission-control-config-file.
// tls.RequireAndVerifyClientCert would fail that handshake and, because these
// webhooks use failurePolicy Fail, would reject every KubeVirt CR write.
// A NetworkPolicy also cannot portably select kube-apiserver: it is usually a
// hostNetwork process, or it is outside the cluster entirely.
//
// Callers are rejected only when their source address is the IP of a
// non-hostNetwork pod outside kube-system. kube-apiserver on the node shows
// up as a node IP. Managed control planes show up as an address that is not a
// pod IP. konnectivity-agent and an in-cluster API server run in kube-system,
// so those pod IPs stay allowed. HostNetwork pod IPs are node IPs and stay
// allowed for the same reason.
//
// Until the pod informer has synced, calls are allowed. Failing closed during
// that window would wedge KubeVirt CR admission on every operator restart.
type PodSourceGuard struct {
	hasSynced func() bool
	listPods  func() []*corev1.Pod
}

// NewPodSourceGuard watches pods and rejects webhook calls whose source
// address belongs to an ordinary pod. A nil client, or a failure to configure
// the informer, returns a guard that allows every caller.
func NewPodSourceGuard(client kubernetes.Interface, stop <-chan struct{}) *PodSourceGuard {
	if client == nil {
		return allowAllPodSourceGuard()
	}

	informer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything()),
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)
	if err := informer.SetTransform(trimWebhookSourcePod); err != nil {
		log.Log.Reason(err).Error("failed to configure virt-operator webhook source guard; pod clients are not filtered")
		return allowAllPodSourceGuard()
	}

	go informer.Run(stop)
	go func() {
		if !cache.WaitForCacheSync(stop, informer.HasSynced) {
			log.Log.Warning("virt-operator webhook source guard did not sync; pod clients are not filtered")
			return
		}
		log.Log.Info("virt-operator webhook is rejecting calls from ordinary pod IPs")
	}()

	return &PodSourceGuard{
		hasSynced: informer.HasSynced,
		listPods: func() []*corev1.Pod {
			return podsFromStore(informer.GetStore())
		},
	}
}

func allowAllPodSourceGuard() *PodSourceGuard {
	return &PodSourceGuard{
		hasSynced: func() bool { return false },
		listPods:  func() []*corev1.Pod { return nil },
	}
}

// Allows reports whether a webhook request from remoteAddr may be served.
// remoteAddr is http.Request.RemoteAddr, host:port or a bare IP.
func (g *PodSourceGuard) Allows(remoteAddr string) bool {
	if g == nil || g.hasSynced == nil || !g.hasSynced() || g.listPods == nil {
		return true
	}

	host := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsedHost
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}

	for _, pod := range g.listPods() {
		if podSourceDenied(pod, ip) {
			return false
		}
	}
	return true
}

// Handler wraps next and rejects callers that Allows denies.
func (g *PodSourceGuard) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.Allows(r.RemoteAddr) {
			log.Log.Warningf("rejected virt-operator webhook request from %s", r.RemoteAddr)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func podSourceDenied(pod *corev1.Pod, ip net.IP) bool {
	if pod == nil || pod.Spec.HostNetwork || pod.Namespace == metav1.NamespaceSystem {
		return false
	}
	for _, podIP := range podIPs(pod) {
		if podIP.Equal(ip) {
			return true
		}
	}
	return false
}

func podIPs(pod *corev1.Pod) []net.IP {
	var ips []net.IP
	if len(pod.Status.PodIPs) > 0 {
		for _, podIP := range pod.Status.PodIPs {
			if ip := net.ParseIP(podIP.IP); ip != nil {
				ips = append(ips, ip)
			}
		}
		return ips
	}
	if ip := net.ParseIP(pod.Status.PodIP); ip != nil {
		ips = append(ips, ip)
	}
	return ips
}

func podsFromStore(store cache.Store) []*corev1.Pod {
	objs := store.List()
	pods := make([]*corev1.Pod, 0, len(objs))
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if ok {
			pods = append(pods, pod)
		}
	}
	return pods
}

// trimWebhookSourcePod drops pod fields the source guard does not read so the
// cluster-wide watch does not retain full pod specs.
func trimWebhookSourcePod(obj interface{}) (interface{}, error) {
	if deleted, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = deleted.Obj
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod == nil {
		return obj, nil
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Spec: corev1.PodSpec{
			HostNetwork: pod.Spec.HostNetwork,
		},
		Status: corev1.PodStatus{
			PodIP:  pod.Status.PodIP,
			PodIPs: pod.Status.PodIPs,
		},
	}, nil
}
