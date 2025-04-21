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

package libinfra

import (
	"context"
	"crypto/x509"
	"reflect"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
)

func ContainsCrt(bundle []byte, containedCrt []byte) bool {
	crts, err := cert.ParseCertsPEM(bundle)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	attached := false
	for _, crt := range crts {
		crtBytes := cert.EncodeCertPEM(crt)
		if reflect.DeepEqual(crtBytes, containedCrt) {
			attached = true
			break
		}
	}
	return attached
}

func GetBundleFromConfigMap(ctx context.Context, configMapName string) ([]byte, []*x509.Certificate) {
	virtClient := kubevirt.Client()
	configMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(ctx, configMapName, v1.GetOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if rawBundle, ok := configMap.Data[components.CABundleKey]; ok {
		crts, err := cert.ParseCertsPEM([]byte(rawBundle))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		return []byte(rawBundle), crts
	}
	return nil, nil
}

// EnsurePodsCertIsSynced waits until new certificates are rolled out to all pods
// that are matching the specified labelselector.
// Once all certificates are in sync, the final secret is returned
func EnsurePodsCertIsSynced(labelSelector string, namespace string, port string) []byte {
	var certs [][]byte
	gomega.EventuallyWithOffset(1, func(g gomega.Gomega) bool {
		var err error
		certs, err = libpod.GetCertsForPods(labelSelector, namespace, port)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		if len(certs) == 0 {
			return true
		}
		for _, crt := range certs[1:] {
			if !reflect.DeepEqual(certs[0], crt) {
				return false
			}
		}
		return true
	}).WithTimeout(90*time.Second).WithPolling(time.Second).Should(gomega.BeTrue(), "certificates across '%s' pods are not in sync", labelSelector)
	if len(certs) > 0 {
		return certs[0]
	}
	return nil
}
