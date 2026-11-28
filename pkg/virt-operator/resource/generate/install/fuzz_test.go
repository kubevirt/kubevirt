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

package install

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func FuzzLoadInstallStrategyFromCache(f *testing.F) {
	validManifest := []byte(`apiVersion: v1
kind: Service
metadata:
  name: fuzz-service
  namespace: kubevirt
spec:
  ports:
  - port: 443
`)
	f.Add(validManifest, uint8(0))
	f.Add(validManifest, uint8(1))
	f.Add([]byte("not-gzip-data"), uint8(2))
	f.Add([]byte("kind: ["), uint8(0))

	config := operatorutil.GetTargetConfigFromKV(&v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: "registry.example.test/kubevirt",
			ImageTag:      "fuzz",
		},
	})

	for _, mode := range []uint8{0, 1} {
		strategy, err := loadInstallStrategyForFuzz(validManifest, mode, config)
		if err != nil {
			f.Fatalf("known-valid install strategy seed in mode %d was rejected: %v", mode, err)
		}
		if strategy == nil {
			f.Fatalf("known-valid install strategy seed in mode %d returned nil", mode)
		}
	}

	f.Fuzz(func(t *testing.T, data []byte, mode uint8) {
		strategy, err := loadInstallStrategyForFuzz(data, mode, config)
		if err != nil {
			return
		}
		if strategy == nil {
			t.Fatal("install strategy load succeeded with a nil strategy")
		}
	})
}

func loadInstallStrategyForFuzz(data []byte, mode uint8, config *operatorutil.KubeVirtDeploymentConfig) (*Strategy, error) {
	annotations := map[string]string{
		v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
		v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
		v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
	}
	manifests := string(data)
	switch mode % 3 {
	case 1:
		var err error
		manifests, err = encodeManifests(data)
		if err != nil {
			return nil, err
		}
		annotations[v1.InstallStrategyConfigMapEncoding] = ManifestsEncodingGzipBase64
	case 2:
		annotations[v1.InstallStrategyConfigMapEncoding] = ManifestsEncodingGzipBase64
	}

	stores := operatorutil.Stores{
		InstallStrategyConfigMapCache: cache.NewStore(cache.MetaNamespaceKeyFunc),
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "fuzz-install-strategy",
			Namespace:   config.GetNamespace(),
			Annotations: annotations,
		},
		Data: map[string]string{
			"manifests": manifests,
		},
	}
	if err := stores.InstallStrategyConfigMapCache.Add(configMap); err != nil {
		return nil, err
	}

	return LoadInstallStrategyFromCache(stores, config)
}
