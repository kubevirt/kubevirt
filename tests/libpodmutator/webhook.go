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

// Package libpodmutator provides helpers for deploying the test-pod-mutator webhook
// in functional tests. The webhook can inject volumes or envFrom ConfigMap references
// into virt-launcher pods, which is useful for overriding runtime configuration via
// environment variables without extending the KubeVirt API.
package libpodmutator

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/pkg/pointer"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// Webhook holds resources created by Setup.
type Webhook struct {
	Pod     *k8sv1.Pod
	Service *k8sv1.Service
	Config  *admissionregistrationv1.MutatingWebhookConfiguration
}

// VolumeType selects the volume source injected by test-pod-mutator.
type VolumeType string

const (
	VolumeTypeEmptyDir  VolumeType = "emptydir"
	VolumeTypeConfigMap VolumeType = "configmap"
)

// VolumeInjection configures a test volume injected into virt-launcher pods.
type VolumeInjection struct {
	Type          VolumeType
	ConfigMapName string // required when Type is VolumeTypeConfigMap
}

// Options configures a test-pod-mutator deployment.
type Options struct {
	Name       string
	SecretName string
	Port       int32
	// Namespace scopes the webhook and MutatingWebhookConfiguration. VMIs whose
	// virt-launcher pods should be mutated must run in this namespace.
	Namespace string
	// EnvFromConfigMap injects the named ConfigMap as envFrom into the compute container.
	EnvFromConfigMap string
	// VolumeInjection injects a test volume into virt-launcher pods.
	VolumeInjection *VolumeInjection
}

func (opts Options) namespace() string {
	if opts.Namespace != "" {
		return opts.Namespace
	}
	return testsuite.GetTestNamespace(nil)
}

func (opts Options) webhookArgs() []string {
	args := []string{fmt.Sprintf("--port=%d", opts.Port)}
	if opts.EnvFromConfigMap != "" {
		args = append(args, fmt.Sprintf("--env-from-configmap=%s", opts.EnvFromConfigMap))
	}
	if opts.VolumeInjection != nil {
		args = append(args, fmt.Sprintf("--volume-type=%s", opts.VolumeInjection.Type))
		if opts.VolumeInjection.Type == VolumeTypeConfigMap {
			args = append(args, fmt.Sprintf("--configmap-name=%s", opts.VolumeInjection.ConfigMapName))
		}
	}
	return args
}

// Setup deploys test-pod-mutator and registers a MutatingWebhookConfiguration scoped
// to the current test namespace.
func Setup(opts Options) *Webhook {
	if opts.VolumeInjection != nil && opts.VolumeInjection.Type == VolumeTypeConfigMap && opts.VolumeInjection.ConfigMapName == "" {
		Expect(false).To(BeTrue(), "VolumeInjection.ConfigMapName is required when Type is VolumeTypeConfigMap")
	}

	cleanupStaleWebhookResources(opts)

	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()

	By("Generating TLS certificates for webhook")
	certPEM, keyPEM, caBundlePEM, err := libinfra.GenerateWebhookCertificates(opts.Name, testNamespace, 24*time.Hour)
	Expect(err).ToNot(HaveOccurred())

	By("Creating secret with webhook certificates")
	secret := libsecret.New(opts.SecretName, libsecret.DataBytes{
		"tls.crt": certPEM,
		"tls.key": keyPEM,
	})
	_, err = virtClient.CoreV1().Secrets(testNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Deploying test-pod-mutator webhook")
	webhookPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: testNamespace,
			Labels: map[string]string{
				"app": opts.Name,
			},
		},
		Spec: k8sv1.PodSpec{
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsNonRoot: pointer.P(true),
				SeccompProfile: &k8sv1.SeccompProfile{
					Type: k8sv1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Containers: []k8sv1.Container{{
				Name:            opts.Name,
				Image:           fmt.Sprintf("%s/test-helpers:%s", flags.KubeVirtRepoPrefix, flags.KubeVirtVersionTag),
				ImagePullPolicy: k8sv1.PullAlways,
				Command:         []string{"/usr/bin/test-pod-mutator"},
				Args:            opts.webhookArgs(),
				Ports: []k8sv1.ContainerPort{{
					ContainerPort: opts.Port,
					Name:          "https",
				}},
				VolumeMounts: []k8sv1.VolumeMount{{
					Name:      "certs",
					MountPath: "/etc/webhook/certs",
					ReadOnly:  true,
				}},
				SecurityContext: &k8sv1.SecurityContext{
					AllowPrivilegeEscalation: pointer.P(false),
					Capabilities: &k8sv1.Capabilities{
						Drop: []k8sv1.Capability{"ALL"},
					},
					RunAsNonRoot: pointer.P(true),
					SeccompProfile: &k8sv1.SeccompProfile{
						Type: k8sv1.SeccompProfileTypeRuntimeDefault,
					},
				},
				ReadinessProbe: &k8sv1.Probe{
					ProbeHandler: k8sv1.ProbeHandler{
						HTTPGet: &k8sv1.HTTPGetAction{
							Path:   "/health",
							Port:   intstr.FromInt(int(opts.Port)),
							Scheme: k8sv1.URISchemeHTTPS,
						},
					},
					InitialDelaySeconds: 1,
					PeriodSeconds:       1,
				},
			}},
			Volumes: []k8sv1.Volume{{
				Name: "certs",
				VolumeSource: k8sv1.VolumeSource{
					Secret: &k8sv1.SecretVolumeSource{
						SecretName: opts.SecretName,
					},
				},
			}},
		},
	}
	webhookPod, err = virtClient.CoreV1().Pods(testNamespace).Create(context.Background(), webhookPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	waitPodReady(opts)

	By("Creating service for webhook")
	webhookService := &k8sv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: testNamespace,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				"app": opts.Name,
			},
			Ports: []k8sv1.ServicePort{{
				Port:       opts.Port,
				TargetPort: intstr.FromInt(int(opts.Port)),
				Name:       "https",
			}},
		},
	}
	webhookService, err = virtClient.CoreV1().Services(testNamespace).Create(context.Background(), webhookService, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	waitServiceEndpointsReady(opts)

	By("Creating MutatingWebhookConfiguration with CA bundle")
	failPolicy := admissionregistrationv1.Fail
	sideEffects := admissionregistrationv1.SideEffectClassNone
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", opts.Name, testNamespace),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name: fmt.Sprintf("%s.kubevirt.io", opts.Name),
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caBundlePEM,
				Service: &admissionregistrationv1.ServiceReference{
					Namespace: testNamespace,
					Name:      opts.Name,
					Path:      pointer.P("/mutate"),
					Port:      pointer.P(opts.Port),
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
				},
			}},
			FailurePolicy: &failPolicy,
			SideEffects:   &sideEffects,
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": testNamespace,
				},
			},
			AdmissionReviewVersions: []string{"v1"},
		}},
	}
	webhookConfig, err = virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), webhookConfig, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	waitAdmissionWebhookReady(opts, testNamespace)

	return &Webhook{
		Pod:     webhookPod,
		Service: webhookService,
		Config:  webhookConfig,
	}
}

// Teardown removes webhook resources created by Setup.
func Teardown(webhook *Webhook, secretName string) {
	if webhook == nil {
		return
	}
	virtClient := kubevirt.Client()
	testNamespace := testsuite.GetTestNamespace(nil)

	if webhook.Config != nil {
		err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), webhook.Config.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		waitMutatingWebhookConfigurationAbsent(webhook.Config.Name)
	}
	if webhook.Service != nil {
		err := virtClient.CoreV1().Services(testNamespace).Delete(context.Background(), webhook.Service.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	if webhook.Pod != nil {
		err := virtClient.CoreV1().Pods(testNamespace).Delete(context.Background(), webhook.Pod.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	err := virtClient.CoreV1().Secrets(testNamespace).Delete(context.Background(), secretName, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func cleanupStaleWebhookResources(opts Options) {
	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()
	mwhName := fmt.Sprintf("%s-%s", opts.Name, testNamespace)

	_ = virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), mwhName, metav1.DeleteOptions{})
	_ = virtClient.CoreV1().Services(testNamespace).Delete(context.Background(), opts.Name, metav1.DeleteOptions{})
	_ = virtClient.CoreV1().Pods(testNamespace).Delete(context.Background(), opts.Name, metav1.DeleteOptions{})
	_ = virtClient.CoreV1().Secrets(testNamespace).Delete(context.Background(), opts.SecretName, metav1.DeleteOptions{})

	waitMutatingWebhookConfigurationAbsent(mwhName)
	waitServiceAbsent(testNamespace, opts.Name)
	Eventually(func() bool {
		_, err := virtClient.CoreV1().Pods(testNamespace).Get(context.Background(), opts.Name, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, 60*time.Second, time.Second).Should(BeTrue())
}

func waitPodReady(opts Options) {
	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()

	By("Waiting for webhook pod to be ready")
	Eventually(func() bool {
		pod, err := virtClient.CoreV1().Pods(testNamespace).Get(context.Background(), opts.Name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == k8sv1.PodReady && cond.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, 60*time.Second, time.Second).Should(BeTrue(), "Webhook pod should be ready")
}

func waitServiceEndpointsReady(opts Options) {
	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()

	By("Waiting for service endpoints to be ready")
	Eventually(func() bool {
		slices, err := virtClient.DiscoveryV1().EndpointSlices(testNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", opts.Name),
		})
		if err != nil {
			return false
		}
		for _, slice := range slices.Items {
			for _, endpoint := range slice.Endpoints {
				if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
					return true
				}
			}
		}
		return false
	}, 30*time.Second, time.Second).Should(BeTrue(), "Service endpoints should be ready")
}

func waitMutatingWebhookConfigurationAbsent(name string) {
	virtClient := kubevirt.Client()
	Eventually(func() bool {
		_, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), name, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, 60*time.Second, time.Second).Should(BeTrue(), "MutatingWebhookConfiguration %s should be deleted", name)
}

func waitServiceAbsent(namespace, name string) {
	virtClient := kubevirt.Client()
	Eventually(func() bool {
		_, err := virtClient.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, 30*time.Second, time.Second).Should(BeTrue(), "Service %s/%s should be deleted", namespace, name)
}

func isWebhookConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "failed calling webhook") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no endpoints available")
}

// waitAdmissionWebhookReady creates a probe pod so Setup blocks until the apiserver
// can successfully call the mutating webhook, not just until the pod is Ready.
func waitAdmissionWebhookReady(opts Options, namespace string) {
	virtClient := kubevirt.Client()
	probeName := fmt.Sprintf("%s-admission-probe", opts.Name)

	By("Verifying the apiserver can reach the admission webhook")
	Eventually(func(g Gomega) {
		_ = virtClient.CoreV1().Pods(namespace).Delete(context.Background(), probeName, metav1.DeleteOptions{})

		_, err := virtClient.CoreV1().Pods(namespace).Create(context.Background(), &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      probeName,
				Namespace: namespace,
			},
			Spec: k8sv1.PodSpec{
				RestartPolicy: k8sv1.RestartPolicyNever,
				Containers: []k8sv1.Container{{
					Name:    "probe",
					Image:   cd.ContainerDiskFor(cd.ContainerDiskAlpine),
					Command: []string{"/bin/sh", "-c", "exit 0"},
				}},
			},
		}, metav1.CreateOptions{})
		if err != nil && isWebhookConnectionError(err) {
			g.Expect(false).To(BeTrue(), "admission webhook is not reachable yet: %v", err)
			return
		}
		g.Expect(err).ToNot(HaveOccurred())
	}, 90*time.Second, 2*time.Second).Should(Succeed())

	_ = virtClient.CoreV1().Pods(namespace).Delete(context.Background(), probeName, metav1.DeleteOptions{})
}
