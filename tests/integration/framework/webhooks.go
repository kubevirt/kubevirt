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
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	mutatingwebhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook"
	validatingwebhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const webhookReadHeaderTimeout = 10 * time.Second

// WithWebhooks enables mutating and validating admission for VirtualMachines.
func WithWebhooks() Option {
	return func(f *Framework) {
		f.webhooksEnabled = true
		f.env.WebhookInstallOptions = envtest.WebhookInstallOptions{
			MutatingWebhooks:   filterMutatingWebhooks(components.NewVirtAPIMutatingWebhookConfiguration(testNamespace)),
			ValidatingWebhooks: filterValidatingWebhooks(components.NewVirtAPIValidatingWebhookConfiguration(testNamespace)),
		}
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
