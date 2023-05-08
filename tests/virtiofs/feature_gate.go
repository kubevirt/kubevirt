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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virtiofs

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] VirtIO-FS feature gate", Serial, decorators.SigStorage, func() {
	var virtClient kubecli.KubevirtClient
	var featureGateWasEnabled bool

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		featureGateWasEnabled = checks.HasFeature(virtconfig.VirtIOFSGate)
		tests.DisableFeatureGate(virtconfig.VirtIOFSGate)
	})

	AfterEach(func() {
		if featureGateWasEnabled {
			tests.EnableFeatureGate(virtconfig.VirtIOFSGate)
		}
	})

	Context("[Serial]With feature gates disabled for", func() {
		It("DataVolume, it should fail to start a VMI", func() {
			vmi := libvmi.NewFedora(libvmi.WithFilesystemDV("something"))
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("virtiofs feature gate is not enabled"))
		})

		It("config volumes, it should success to start a VMI", func() {
			configMapName := "configmap-" + uuid.NewRandom().String()[:6]
			configMapData := map[string]string{
				"option1": "value1",
				"option2": "value2",
				"option3": "value3",
			}
			tests.CreateConfigMap(configMapName, testsuite.GetTestNamespace(nil), configMapData)

			secretName := "secret-" + uuid.NewRandom().String()[:6]
			secretData := map[string]string{
				"user":     "admin",
				"password": "redhat",
			}
			tests.CreateSecret(secretName, testsuite.GetTestNamespace(nil), secretData)

			serviceAccountVolumeName := "default-disk"

			downwardAPIName := "downwardapi-" + uuid.NewRandom().String()[:6]
			testLabelKey := "kubevirt.io.testdownwardapi"
			testLabelVal := "downwardAPIValue"

			By("Creating VMI")
			vmi := libvmi.NewFedora(
				libvmi.WithConfigMapFs(configMapName, configMapName),
				libvmi.WithSecretFs(secretName, secretName),
				libvmi.WithServiceAccountFs("default", serviceAccountVolumeName),
				libvmi.WithLabel(testLabelKey, testLabelVal),
				libvmi.WithDownwardAPIFs(downwardAPIName),
			)

			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("DataVolume and ServiceAccount, it should fail to start a VMI", func() {
			serviceAccountVolumeName := "default-disk"
			vmi := libvmi.NewFedora(
				libvmi.WithFilesystemDV("something"),
				libvmi.WithServiceAccountFs("default", serviceAccountVolumeName),
			)
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("virtiofs feature gate is not enabled"))
		})
	})
})
