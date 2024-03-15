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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libinstancetype

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/testsuite"
)

func CheckForVMInstancetypeRevisionNames(vmName string, virtClient kubecli.KubevirtClient) func() error {
	return func() error {
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if vm.Spec.Instancetype.RevisionName == "" {
			return fmt.Errorf("instancetype revision name is expected to not be empty")
		}

		if vm.Spec.Preference.RevisionName == "" {
			return fmt.Errorf("preference revision name is expected to not be empty")
		}
		return nil
	}
}

func WaitForVMInstanceTypeRevisionNames(vmName string, virtClient kubecli.KubevirtClient) {
	Eventually(CheckForVMInstancetypeRevisionNames(vmName, virtClient), 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func EnsureControllerRevisionObjectsEqual(crNameA, crNameB string, virtClient kubecli.KubevirtClient) bool {
	crA, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameA, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	crB, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameB, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return equality.Semantic.DeepEqual(crA.Data.Object, crB.Data.Object)
}
