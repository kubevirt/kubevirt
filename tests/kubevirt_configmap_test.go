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

package tests_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = Describe("[Serial]KubeVirtConfigmapConfiguration", func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		cfgMap := &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: virtconfig.ConfigMapName},
			Data: map[string]string{
				virtconfig.FeatureGatesKey:        "CPUManager, LiveMigration, ExperimentalIgnitionSupport, Sidecar, Snapshot",
				virtconfig.SELinuxLauncherTypeKey: "virt_launcher.process",
			},
		}

		_, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Create(context.Background(), cfgMap, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Delete(context.Background(), virtconfig.ConfigMapName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:4670]check health check returns configmap resource version", func() {
		cfg, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), virtconfig.ConfigMapName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		tests.WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-controller", cfg.ResourceVersion, tests.ExpectResourceVersionToBeEqualConfigVersion)
		tests.WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-api", cfg.ResourceVersion, tests.ExpectResourceVersionToBeEqualConfigVersion)
		tests.WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-handler", cfg.ResourceVersion, tests.ExpectResourceVersionToBeEqualConfigVersion)
	})

	// TODO config-map test overide my kv feature gate
	It("[test_id:4671]test kubevirt config-map is used for configuration when present", func() {

		vmi := tests.NewRandomFedoraVMIWithDmidecode()

		test_smbios := &v1.SMBiosConfiguration{Family: "configmap", Product: "test", Manufacturer: "None", Sku: "2.0", Version: "2.0"}
		smbiosJson, err := json.Marshal(test_smbios)
		Expect(err).ToNot(HaveOccurred())
		tests.UpdateClusterConfigValueAndWait(virtconfig.SmbiosConfigKey, string(smbiosJson))

		By("Starting a VirtualMachineInstance")
		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())
		tests.WaitForSuccessfulVMIStart(vmi)

		domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(domXml).To(ContainSubstring("<entry name='family'>configmap</entry>"))
		Expect(domXml).To(ContainSubstring("<entry name='product'>test</entry>"))
		Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>None</entry>"))
		Expect(domXml).To(ContainSubstring("<entry name='sku'>2.0</entry>"))
		Expect(domXml).To(ContainSubstring("<entry name='version'>2.0</entry>"))
	})
})
