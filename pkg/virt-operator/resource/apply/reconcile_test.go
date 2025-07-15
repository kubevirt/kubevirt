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

package apply

import (
	"bufio"
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	marshalutil "kubevirt.io/kubevirt/tools/util"
)

type MockStore struct {
	get interface{}
}

func (m *MockStore) Add(_ interface{}) error    { return nil }
func (m *MockStore) Update(_ interface{}) error { return nil }
func (m *MockStore) Delete(_ interface{}) error { return nil }
func (m *MockStore) List() []interface{}        { return nil }
func (m *MockStore) ListKeys() []string         { return nil }
func (m *MockStore) Get(_ interface{}) (item interface{}, exists bool, err error) {
	item = m.get
	if m.get != nil {
		exists = true
	}
	return
}
func (m *MockStore) GetByKey(_ string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}
func (m *MockStore) Replace([]interface{}, string) error { return nil }
func (m *MockStore) Resync() error                       { return nil }

const (
	Namespace = "ns"
	Version   = "1.0"
	Registry  = "rep"
	Id        = "42"
)

func getConfig(registry, version string) *util.KubeVirtDeploymentConfig {
	return util.GetTargetConfigFromKV(&v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: registry,
			ImageTag:      version,
		},
	})
}

func loadTargetStrategy(resource interface{}, config *util.KubeVirtDeploymentConfig, stores util.Stores) *install.Strategy {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	marshalutil.MarshallObject(resource, writer)
	writer.Flush()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    config.GetNamespace(),
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
			},
		},
		Data: map[string]string{
			"manifests": string(b.Bytes()),
		},
	}

	stores.InstallStrategyConfigMapCache.Add(configMap)
	targetStrategy, err := installstrategy.LoadInstallStrategyFromCache(stores, config)
	Expect(err).ToNot(HaveOccurred())

	return targetStrategy
}

var _ = Describe("Apply", func() {

	Context("should calculate", func() {

		DescribeTable("update path based on semver", func(target string, current string, expected bool) {
			takeUpdatePath := shouldTakeUpdatePath(target, current)

			Expect(takeUpdatePath).To(Equal(expected))
		},
			Entry("with increasing semver", "v0.15.0", "v0.14.0", true),
			Entry("with decreasing semver", "v0.14.0", "v0.15.0", false),
			Entry("with identical semver", "v0.15.0", "v0.15.0", false),
			Entry("with invalid semver", "devel", "v0.14.0", true),
			Entry("with increasing semver no prefix", "0.15.0", "0.14.0", true),
			Entry("with decreasing semver no prefix", "0.14.0", "0.15.0", false),
			Entry("with identical semver no prefix", "0.15.0", "0.15.0", false),
			Entry("with invalid semver no prefix", "devel", "0.14.0", true),
			Entry("with no current no prefix", "devel", "", false),
		)
	})

	Context("Injecting Metadata", func() {

		It("should set expected values", func() {

			kv := &v1.KubeVirt{}
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version
			kv.Status.TargetDeploymentID = Id

			deployment := appsv1.Deployment{}
			injectOperatorMetadata(kv, &deployment.ObjectMeta, "fakeversion", "fakeregistry", "fakeid", false)

			// NOTE we are purposfully not using the defined constant values
			// in types.go here. This test is explicitly verifying that those
			// values in types.go that we depend on for virt-operator updates
			// do not change. This is meant to preserve backwards and forwards
			// compatibility

			managedBy, ok := deployment.Labels["app.kubernetes.io/managed-by"]

			Expect(ok).To(BeTrue())
			Expect(managedBy).To(Equal("virt-operator"))

			version, ok := deployment.Annotations["kubevirt.io/install-strategy-version"]
			Expect(ok).To(BeTrue())
			Expect(version).To(Equal("fakeversion"))

			registry, ok := deployment.Annotations["kubevirt.io/install-strategy-registry"]
			Expect(ok).To(BeTrue())
			Expect(registry).To(Equal("fakeregistry"))

			id, ok := deployment.Annotations["kubevirt.io/install-strategy-identifier"]
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal("fakeid"))
		})
	})
})
