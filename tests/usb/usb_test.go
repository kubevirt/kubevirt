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

package usb

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func TestCleanupWithoutCreatedVMI(t *testing.T) {
	client := kubecli.NewMockKubevirtClient(gomock.NewController(t))
	if err := cleanupVMI(client, nil, nil); err != nil {
		t.Fatalf("cleanup after failed setup: %v", err)
	}
}

func TestCleanupCreatedVMI(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := kubecli.NewMockKubevirtClient(ctrl)
	vmis := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Namespace: "usb-test", Name: "guest"}}
	client.EXPECT().VirtualMachineInstance(vmi.Namespace).Return(vmis)
	vmis.EXPECT().Delete(gomock.Any(), vmi.Name, metav1.DeleteOptions{}).Return(nil)

	waited := false
	err := cleanupVMI(client, vmi, func(target *v1.VirtualMachineInstance, timeout time.Duration) error {
		if !ctrl.Satisfied() {
			t.Fatal("wait started before the VMI was deleted")
		}
		if target != vmi || timeout != 180*time.Second {
			t.Fatalf("unexpected wait for %v with timeout %v", target, timeout)
		}
		waited = true
		return nil
	})
	if err != nil || !waited {
		t.Fatalf("cleanup returned %v, waited for deletion: %t", err, waited)
	}
}
