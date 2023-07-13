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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"kubevirt.io/kubevirt/tests/framework/checks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = DescribeInfra("tls configuration", func() {

	var (
		virtClient kubecli.KubevirtClient
		cipher     *tls.CipherSuite
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if !checks.HasFeature(virtconfig.VMExportGate) {
			Skip(fmt.Sprintf("Cluster has the %s featuregate disabled, skipping  the tests", virtconfig.VMExportGate))
		}

		// FIPS-compliant so we can test on different platforms (otherwise won't revert properly)
		cipher = &tls.CipherSuite{
			ID:   tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			Name: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		}
		kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
		kvConfig.TLSConfiguration = &v1.TLSConfiguration{
			MinTLSVersion: v1.VersionTLS12,
			Ciphers:       []string{cipher.Name},
		}
		tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
		newKv := util.GetCurrentKv(virtClient)
		Expect(newKv.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(BeEquivalentTo(v1.VersionTLS12))
		Expect(newKv.Spec.Configuration.TLSConfiguration.Ciphers).To(BeEquivalentTo([]string{cipher.Name}))

	})

	It("[test_id:9306]should result only connections with the correct client-side tls configurations are accepted by the components", func() {
		labelSelectorList := []string{"kubevirt.io=virt-api", "kubevirt.io=virt-handler", "kubevirt.io=virt-exportproxy"}

		var podsToTest []k8sv1.Pod
		for _, labelSelector := range labelSelectorList {
			podList, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			Expect(err).ToNot(HaveOccurred())
			podsToTest = append(podsToTest, podList.Items...)
		}

		for i, pod := range podsToTest {
			func(i int, pod k8sv1.Pod) {
				stopChan := make(chan struct{})
				defer close(stopChan)
				Expect(tests.ForwardPorts(&pod, []string{fmt.Sprintf("844%d:%d", i, 8443)}, stopChan, 10*time.Second)).To(Succeed())

				acceptedTLSConfig := &tls.Config{
					InsecureSkipVerify: true,
					MaxVersion:         tls.VersionTLS12,
					CipherSuites:       kvtls.CipherSuiteIds([]string{cipher.Name}),
				}
				conn, err := tls.Dial("tcp", fmt.Sprintf("localhost:844%d", i), acceptedTLSConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(conn).ToNot(BeNil())
				Expect(conn.ConnectionState().Version).To(BeEquivalentTo(tls.VersionTLS12))
				Expect(conn.ConnectionState().CipherSuite).To(BeEquivalentTo(cipher.ID))

				rejectedTLSConfig := &tls.Config{
					InsecureSkipVerify: true,
					MaxVersion:         tls.VersionTLS11,
				}
				conn, err = tls.Dial("tcp", fmt.Sprintf("localhost:844%d", i), rejectedTLSConfig)
				Expect(err).To(HaveOccurred())
				Expect(conn).To(BeNil())
				Expect(err.Error()).To(SatisfyAny(
					BeEquivalentTo("remote error: tls: protocol version not supported"),
					// The error message changed with the golang 1.19 update
					BeEquivalentTo("tls: no supported versions satisfy MinVersion and MaxVersion"),
				))
			}(i, pod)
		}
	})
})
