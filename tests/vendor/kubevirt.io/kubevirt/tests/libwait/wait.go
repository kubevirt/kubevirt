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

package libwait

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/watcher"
)

// WaitForVMIStartOrFailed blocks until the specified VirtualMachineInstance reached either Failed or Running states
func WaitForVMIStartOrFailed(obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Running, v1.Failed}, obj, seconds, wp, true)
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func WaitForVMIStart(ctx context.Context, obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Running}, obj, seconds, wp, false)
}

// Block until the specified VirtualMachineInstance scheduled and return the target node name.
func WaitForVMIScheduling(obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, obj, seconds, wp, false)
}

func waitForVMIPhase(ctx context.Context, phases []v1.VirtualMachineInstancePhase, obj runtime.Object, seconds int, wp watcher.WarningsPolicy, waitForFail bool) *v1.VirtualMachineInstance {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	gomega.ExpectWithOffset(1, ok).To(gomega.BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())

	// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
	// FIXME: This may start watching too late and we may miss some warnings
	if vmi.ResourceVersion == "" {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	}

	objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(seconds+2) * time.Second)
	if wp.FailOnWarnings == true {
		// let's ignore PSA events as kubernetes internally uses a namespace informer
		// that might not be up to date after virt-controller relabeled the namespace
		// to use a 'privileged' policy
		// TODO: remove this when KubeVirt will be able to run VMs under the 'restricted' level
		wp.WarningsIgnoreList = append(wp.WarningsIgnoreList, "violates PodSecurity")
		objectEventWatcher.SetWarningsPolicy(wp)
	}

	go func() {
		defer ginkgo.GinkgoRecover()
		objectEventWatcher.WaitFor(ctx, watcher.NormalEvent, v1.Started)
	}()

	timeoutMsg := fmt.Sprintf("Timed out waiting for VMI %s to enter %s phase(s)", vmi.Name, phases)
	// FIXME the event order is wrong. First the document should be updated
	gomega.EventuallyWithOffset(1, func(g gomega.Gomega) v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		g.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())

		g.Expect(vmi).ToNot(matcher.HaveSucceeded(), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		// May need to wait for Failed state
		if !waitForFail {
			g.Expect(vmi).ToNot(matcher.BeInPhase(v1.Failed), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		}
		return vmi.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(gomega.BeElementOf(phases), timeoutMsg)

	return vmi
}

func WaitForSuccessfulVMIStartIgnoreWarnings(vmi runtime.Object) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: false}
	return WaitForVMIStart(ctx, vmi, 180, wp)
}

func WaitForSuccessfulVMIStartWithTimeout(vmi runtime.Object, seconds int) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return WaitForVMIStart(ctx, vmi, seconds, wp)
}

func WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi runtime.Object, seconds int) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: false}
	return WaitForVMIStart(ctx, vmi, seconds, wp)
}

func WaitForVirtualMachineToDisappearWithTimeout(vmi *v1.VirtualMachineInstance, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	gomega.EventuallyWithOffset(1, func() error {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		return err
	}, seconds, 1*time.Second).Should(gomega.SatisfyAll(gomega.HaveOccurred(), gomega.WithTransform(errors.IsNotFound, gomega.BeTrue())), "The VMI should be gone within the given timeout")
}

func WaitForMigrationToDisappearWithTimeout(migration *v1.VirtualMachineInstanceMigration, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	gomega.EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("migration %s was expected to dissapear after %d seconds, but it did not", migration.Name, seconds))
}

func WaitForSuccessfulVMIStart(vmi runtime.Object) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitForSuccessfulVMIStartWithContext(ctx, vmi)
}

func WaitUntilVMIReady(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitUntilVMIReadyWithContext(ctx, vmi, loginTo)
}

func WaitUntilVMIReadyIgnoreSelectedWarnings(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitUntilVMIReadyWithContextIgnoreSelectedWarnings(ctx, vmi, loginTo, warningsIgnoreList)
}

func WaitUntilVMIReadyWithContext(ctx context.Context, vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	vmi = WaitForSuccessfulVMIStartWithContext(ctx, vmi)
	gomega.Expect(loginTo(vmi)).To(gomega.Succeed())
	return vmi
}

func WaitUntilVMIReadyWithContextIgnoreSelectedWarnings(ctx context.Context, vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	vmi = WaitForSuccessfulVMIStartWithContextIgnoreSelectedWarnings(ctx, vmi, warningsIgnoreList)
	gomega.Expect(loginTo(vmi)).To(gomega.Succeed())
	return vmi
}

func WaitForSuccessfulVMIStartWithContext(ctx context.Context, vmi runtime.Object) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return WaitForVMIStart(ctx, vmi, 360, wp)
}

func WaitForSuccessfulVMIStartWithContextIgnoreSelectedWarnings(ctx context.Context, vmi runtime.Object, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: warningsIgnoreList}
	return WaitForVMIStart(ctx, vmi, 360, wp)
}
