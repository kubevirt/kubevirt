/*
 * This file is part of the kubevirt project
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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	defaultPodNetworkName  = "default"
	linuxBridgeIfaceName1  = "nic1"
	linuxBridgeIfaceName2  = "nic2"
	linuxBridgeNetworkName = "linux-br"
	bridgeCNIType          = "bridge"
	bridgeName             = "br10"

	linuxBridgeNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"%s\", \"bridge\": \"%s\"}]}"}}`
)

var _ = SIGDescribe("kubectl", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	getVmiHeader := []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "READY"}

	createBridgeNetworkAttachmentDefinition := func(namespace, networkName string, bridgeCNIType string, bridgeName string) error {
		bridgeNad := fmt.Sprintf(linuxBridgeNAD, networkName, namespace, bridgeCNIType, bridgeName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, bridgeNad)
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("should verify vmi ip value match primary interface ip value", func() {
		Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), linuxBridgeNetworkName, bridgeCNIType, bridgeName)).
			To(Succeed())

		vmi := libvmi.New(
			libvmi.WithResourceMemory("2Mi"),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeIfaceName1, linuxBridgeNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeIfaceName2, linuxBridgeNetworkName)),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeIfaceName1)),
			libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeIfaceName2)),
		)

		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitForSuccessfulVMIStart(vmi)

		primaryIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, defaultPodNetworkName)
		Expect(primaryIfaceStatus).ToNot(BeNil())

		var result, errStr string
		pollErr := wait.PollImmediate(time.Second*1, time.Second*5, func() (bool, error) {
			result, errStr, err = clientcmd.RunCommand(clientcmd.GetK8sCmdClient(), "get", "vmi", vmi.Name)
			return err == nil, nil
		})

		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error: %s", errStr))
		Expect(pollErr).ToNot(HaveOccurred())
		Expect(result).ToNot(BeEmpty())
		resultFields := strings.Fields(result)

		Expect(resultFields[getVMIHeaderIPIndex(getVmiHeader)]).To(Equal(primaryIfaceStatus.IP), "should match primary interface ip")
	})
})

func getVMIHeaderIPIndex(getVmiHeader []string) int {
	const ipPosition = 3

	return len(getVmiHeader) + ipPosition
}
