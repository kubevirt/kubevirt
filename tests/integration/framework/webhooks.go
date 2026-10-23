/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package framework

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	mutatingwebhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook"
	validatingwebhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	webhookReadHeaderTimeout = 10 * time.Second
	webhookShutdownTimeout   = 10 * time.Second
	webhookAdmissionTimeout  = time.Minute
	webhookAdmissionInterval = 100 * time.Millisecond
)

// WithoutWebhooks disables VM admission webhooks to isolate API server schema and CEL validation.
// VM webhooks are enabled by default.
func WithoutWebhooks() Option {
	return func(f *Framework) {
		f.webhooksEnabled = false
	}
}

func (f *Framework) configureWebhooks() {
	f.env.WebhookInstallOptions = envtest.WebhookInstallOptions{
		MutatingWebhooks:   filterMutatingWebhooks(components.NewVirtAPIMutatingWebhookConfiguration(testNamespace)),
		ValidatingWebhooks: filterValidatingWebhooks(components.NewVirtAPIValidatingWebhookConfiguration(testNamespace)),
	}
}

func (f *Framework) startWebhookServer(clusterConfig *virtconfig.ClusterConfig, factory controller.KubeInformerFactory) {
	GinkgoHelper()
	opts := f.env.WebhookInstallOptions
	cert, err := tls.LoadX509KeyPair(
		filepath.Join(opts.LocalServingCertDir, "tls.crt"),
		filepath.Join(opts.LocalServingCertDir, "tls.key"),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to load webhook TLS certs")

	webhookInformers := &webhooks.Informers{
		DataSourceInformer: factory.DummyDataSource(),
		NamespaceInformer:  factory.Namespace(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(components.VMMutatePath, func(w http.ResponseWriter, r *http.Request) {
		mutatingwebhook.ServeVMs(w, r, clusterConfig, f.virtClient)
	})
	mux.HandleFunc(components.VMValidatePath, func(w http.ResponseWriter, r *http.Request) {
		// Controllers and tests share the envtest admin identity. Keep it subject to
		// regular-user validation; trusting it would also exempt test requests.
		validatingwebhook.ServeVMs(w, r, clusterConfig, f.virtClient, webhookInformers, nil)
	})

	var listenConfig net.ListenConfig
	address := net.JoinHostPort(opts.LocalServingHost, strconv.Itoa(opts.LocalServingPort))
	listener, err := listenConfig.Listen(f.ctx, "tcp", address)
	Expect(err).NotTo(HaveOccurred(), "failed to listen for webhook requests")
	f.webhookServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: webhookReadHeaderTimeout,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}
	f.webhookDone = make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(f.webhookDone)
		err := f.webhookServer.Serve(tls.NewListener(listener, f.webhookServer.TLSConfig))
		if !errors.Is(err, http.ErrServerClosed) {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("webhook server failed on %s", listener.Addr()))
		}
	}()
}

// Configuration discovery alone does not guarantee the API server has observed
// the webhooks. Probe both handlers through admission without persisting a VM.
func (f *Framework) waitForWebhookAdmission() {
	GinkgoHelper()
	ctx, cancel := context.WithTimeout(f.ctx, webhookAdmissionTimeout)
	defer cancel()
	vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithMemoryRequest("128Mi")))
	opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
	Eventually(ctx, func(g Gomega) {
		mutated, err := f.virtClient.VirtualMachine(testNamespace).Create(ctx, vm, opts)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(mutated.Spec.Template.Spec.Domain.Firmware).NotTo(BeNil())
		g.Expect(mutated.Spec.Template.Spec.Domain.Firmware.UUID).NotTo(BeEmpty())

		invalid := vm.DeepCopy()
		invalid.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{{
			Name:       "missing-volume",
			DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
		}}
		_, err = f.virtClient.VirtualMachine(testNamespace).Create(ctx, invalid, opts)
		g.Expect(err).To(MatchError(k8serrors.IsInvalid, "IsInvalid"))
		g.Expect(err).To(MatchError(ContainSubstring("virtualmachine-validator.kubevirt.io")))
		g.Expect(err).To(MatchError(ContainSubstring("missing-volume")))
	}, webhookAdmissionInterval).Should(Succeed(), "VM admission webhooks did not become effective")
}

func (f *Framework) stopWebhookServer() error {
	// The framework context is already canceled; allow active requests a fresh
	// deadline to finish while the API server is still available.
	ctx, cancel := context.WithTimeout(context.Background(), webhookShutdownTimeout)
	defer cancel()
	err := f.webhookServer.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, f.webhookServer.Close())
	}
	<-f.webhookDone
	return err
}

func servesVirtualMachines(rules []admissionregistrationv1.RuleWithOperations) bool {
	return slices.ContainsFunc(rules, func(rule admissionregistrationv1.RuleWithOperations) bool {
		return slices.Contains(rule.Resources, "virtualmachines")
	})
}

func filterMutatingWebhooks(
	cfg *admissionregistrationv1.MutatingWebhookConfiguration,
) []*admissionregistrationv1.MutatingWebhookConfiguration {
	filtered := cfg.DeepCopy()
	filtered.Webhooks = slices.DeleteFunc(filtered.Webhooks, func(wh admissionregistrationv1.MutatingWebhook) bool {
		return !servesVirtualMachines(wh.Rules)
	})
	return []*admissionregistrationv1.MutatingWebhookConfiguration{filtered}
}

func filterValidatingWebhooks(
	cfg *admissionregistrationv1.ValidatingWebhookConfiguration,
) []*admissionregistrationv1.ValidatingWebhookConfiguration {
	filtered := cfg.DeepCopy()
	filtered.Webhooks = slices.DeleteFunc(filtered.Webhooks, func(wh admissionregistrationv1.ValidatingWebhook) bool {
		return !servesVirtualMachines(wh.Rules)
	})
	return []*admissionregistrationv1.ValidatingWebhookConfiguration{filtered}
}
