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

package libnet

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

const (
	guestAgentConnectedTimeout = 6 * time.Minute
	guestAgentConnectedPoll    = 2 * time.Second
	ifaceReportedTimeout       = 2 * time.Minute
	ifaceReportedPoll          = 3 * time.Second
)

// WaitUntilVMINetworkIfaceReportedByGuestAgent waits for VirtualMachineInstanceAgentConnected and a
// Status.Interfaces entry with Name==networkName, guest-agent in InfoSource, and IP or IPs non-empty.
func WaitUntilVMINetworkIfaceReportedByGuestAgent(vmi *v1.VirtualMachineInstance, networkName string) {
	EventuallyWithOffset(1, matcher.ThisVMI(vmi)).
		WithTimeout(guestAgentConnectedTimeout).
		WithPolling(guestAgentConnectedPoll).
		Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	EventuallyWithOffset(1, func() error {
		latest, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for i := range latest.Status.Interfaces {
			iface := &latest.Status.Interfaces[i]
			if iface.Name != networkName {
				continue
			}
			if !vmispec.ContainsInfoSource(iface.InfoSource, vmispec.InfoSourceGuestAgent) {
				return fmt.Errorf("interface %q: guest-agent not in infoSource %q yet", iface.Name, iface.InfoSource)
			}
			if iface.IP == "" && len(iface.IPs) == 0 {
				return fmt.Errorf("interface %q: no IPs in status yet", iface.Name)
			}
			return nil
		}
		return fmt.Errorf("interface for network %q not found in VMI status yet", networkName)
	}).
		WithTimeout(ifaceReportedTimeout).
		WithPolling(ifaceReportedPoll).
		Should(Succeed(), "network interface should be reported by guest agent with IP addresses")
}

// WaitUntilDefaultPodNetworkIfaceReportedByGuestAgent is the same for the default pod network.
func WaitUntilDefaultPodNetworkIfaceReportedByGuestAgent(vmi *v1.VirtualMachineInstance) {
	WaitUntilVMINetworkIfaceReportedByGuestAgent(vmi, v1.DefaultPodNetwork().Name)
}
