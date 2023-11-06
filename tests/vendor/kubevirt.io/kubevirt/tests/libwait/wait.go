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

	"kubevirt.io/kubevirt/tests/testsuite"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/watcher"
)

const defaultTimeout = 360

// Option represents an action that enables an option.
type Option func(waiting *Waiting)

// Waiting represents the waiting struct.
type Waiting struct {
	ctx         context.Context
	wp          *watcher.WarningsPolicy
	timeout     int
	waitForFail bool
	phases      []v1.VirtualMachineInstancePhase
}

// WaitForVMIPhase blocks until the specified VirtualMachineInstance reaches any of the phases.
// By default, the waiting will fail if a warning is captured and a default timeout will be used.
// These properties can be customized using With* options.
// If no context is provided, a new one will be created.
func WaitForVMIPhase(vmi *v1.VirtualMachineInstance, phases []v1.VirtualMachineInstancePhase, opts ...Option) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	gomega.ExpectWithOffset(1, phases).ToNot(gomega.BeEmpty())
	waiting := Waiting{timeout: defaultTimeout, wp: &wp, phases: phases}
	for _, f := range opts {
		f(&waiting)
	}

	if waiting.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		waiting.ctx = ctx
	}

	WithWarningsIgnoreList(testsuite.TestRunConfiguration.WarningToIgnoreList)(&waiting)
	return waiting.watchVMIForPhase(vmi)
}

// WithContext adds a specific context to the waiting struct
func WithContext(ctx context.Context) Option {
	return func(waiting *Waiting) {
		waiting.ctx = ctx
	}
}

// WithTimeout adds a specific timeout to the waiting struct
func WithTimeout(seconds int) Option {
	return func(waiting *Waiting) {
		waiting.timeout = seconds
	}
}

// WithWarningsPolicy adds a specific warningPolicy to the waiting struct
func WithWarningsPolicy(wp *watcher.WarningsPolicy) Option {
	return func(waiting *Waiting) {
		waiting.wp = wp
	}
}

// WithFailOnWarnings sets if the waiting should fail on warning or not
func WithFailOnWarnings(failOnWarnings bool) Option {
	return func(waiting *Waiting) {
		if waiting.wp == nil {
			waiting.wp = &watcher.WarningsPolicy{}
		}

		waiting.wp.FailOnWarnings = failOnWarnings
	}
}

// WithWarningsIgnoreList sets the warnings that will be ignored during the waiting for phase
// This option will be ignored if a warning policy has been set before and the failOnWarnings is false
// If no warning policy has been set before, a new one will be set implicitly with fail on warnings enabled and the ignore list added
func WithWarningsIgnoreList(warningIgnoreList []string) Option {
	return func(waiting *Waiting) {
		if waiting.wp == nil {
			waiting.wp = &watcher.WarningsPolicy{FailOnWarnings: true}
		}

		waiting.wp.WarningsIgnoreList = append(waiting.wp.WarningsIgnoreList, warningIgnoreList...)
	}
}

// WithWaitForFail adds the specific waitForFail to the waiting struct
func WithWaitForFail(waitForFail bool) Option {
	return func(waiting *Waiting) {
		waiting.waitForFail = waitForFail
	}
}

// watchVMIForPhase looks at the vmi object and, after it is started, waits for it to satisfy the phases with the passed parameters
func (w *Waiting) watchVMIForPhase(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())

	// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
	// FIXME: This may start watching too late and we may miss some warnings
	if vmi.ResourceVersion == "" {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	}

	objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(w.timeout+2) * time.Second)
	if w.wp.FailOnWarnings == true {
		// let's ignore PSA events as kubernetes internally uses a namespace informer
		// that might not be up to date after virt-controller relabeled the namespace
		// to use a 'privileged' policy
		// TODO: remove this when KubeVirt will be able to run VMs under the 'restricted' level
		w.wp.WarningsIgnoreList = append(w.wp.WarningsIgnoreList, "violates PodSecurity")
		objectEventWatcher.SetWarningsPolicy(*w.wp)
	}

	go func() {
		defer ginkgo.GinkgoRecover()
		objectEventWatcher.WaitFor(w.ctx, watcher.NormalEvent, v1.Started)
	}()

	timeoutMsg := fmt.Sprintf("Timed out waiting for VMI %s to enter %s phase(s)", vmi.Name, w.phases)
	// FIXME the event order is wrong. First the document should be updated
	gomega.EventuallyWithOffset(1, func(g gomega.Gomega) v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		g.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())

		g.Expect(vmi).ToNot(matcher.HaveSucceeded(), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		// May need to wait for Failed state
		if !w.waitForFail {
			g.Expect(vmi).ToNot(matcher.BeInPhase(v1.Failed), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		}
		return vmi.Status.Phase
	}, time.Duration(w.timeout)*time.Second, 1*time.Second).Should(gomega.BeElementOf(w.phases), timeoutMsg)

	return vmi
}

// WaitForSuccessfulVMIStart blocks until the specified VirtualMachineInstance reaches the Running state
// using the passed options
func WaitForSuccessfulVMIStart(vmi *v1.VirtualMachineInstance, opts ...Option) *v1.VirtualMachineInstance {
	return WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		opts...,
	)
}

// WaitUntilVMIReady blocks until the specified VirtualMachineInstance reaches the Running state using the passed
// options, and the login succeed
func WaitUntilVMIReady(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, opts ...Option) *v1.VirtualMachineInstance {
	vmi = WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		opts...,
	)
	gomega.Expect(loginTo(vmi)).To(gomega.Succeed())
	return vmi
}

// WaitForVirtualMachineToDisappearWithTimeout blocks for the passed seconds until the specified VirtualMachineInstance disappears
func WaitForVirtualMachineToDisappearWithTimeout(vmi *v1.VirtualMachineInstance, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	gomega.EventuallyWithOffset(1, func() error {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		return err
	}, seconds, 1*time.Second).Should(gomega.SatisfyAll(gomega.HaveOccurred(), gomega.WithTransform(errors.IsNotFound, gomega.BeTrue())), "The VMI should be gone within the given timeout")
}

// WaitForMigrationToDisappearWithTimeout blocks for the passed seconds until the specified VirtualMachineInstanceMigration disappears
func WaitForMigrationToDisappearWithTimeout(migration *v1.VirtualMachineInstanceMigration, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	gomega.EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("migration %s was expected to dissapear after %d seconds, but it did not", migration.Name, seconds))
}
