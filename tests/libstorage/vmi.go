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

package libstorage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
)

func LookupVolumeTargetPath(vmi *v1.VirtualMachineInstance, volumeName string) string {
	for _, volStatus := range vmi.Status.VolumeStatus {
		if volStatus.Name == volumeName {
			return fmt.Sprintf("/dev/%s", volStatus.Target)
		}
	}

	return ""
}

func WaitForCBTEnabled(virtClient kubecli.KubevirtClient, namespace, name string) {
	Eventually(func() v1.ChangedBlockTrackingState {
		vm, err := virtClient.VirtualMachine(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		return cbt.CBTState(vm.Status.ChangedBlockTracking)
	}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

	Eventually(func() v1.ChangedBlockTrackingState {
		vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		return cbt.CBTState(vmi.Status.ChangedBlockTracking)
	}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))
}
