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

package vm

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_HIBERNATE           = "hibernate"
	COMMAND_DISCARD_SAVED_STATE = "discard-saved-state"

	storageClassArg = "storage-class"
	waitArg         = "wait"
	timeoutArg      = "timeout"

	hibernatePollInterval = time.Second
)

var (
	storageClass string
	waitForSave  bool
	waitTimeout  time.Duration
)

func NewHibernateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hibernate (VM)",
		Short: "Save the memory and device state of a running virtual machine to a PVC and stop it. The next start resumes it.",
		Example: `  # Hibernate a virtual machine called 'myvm':
  {{ProgramName}} hibernate myvm

  # Hibernate 'myvm' to a PVC of the storage class 'fast' and wait until its state is saved:
  {{ProgramName}} hibernate myvm --storage-class=fast --wait

  # Resume 'myvm' from its saved state:
  {{ProgramName}} start myvm`,
		Args: cobra.ExactArgs(1),
		RunE: hibernateRun,
	}
	cmd.Flags().StringVar(&storageClass, storageClassArg, "", "The storage class of the PVC holding the saved state. Defaults to the default storage class.")
	cmd.Flags().BoolVar(&waitForSave, waitArg, false, "Wait until the state is saved and the virtual machine instance is gone, or the hibernation failed.")
	cmd.Flags().DurationVar(&waitTimeout, timeoutArg, 10*time.Minute, "How long --wait waits.")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewDiscardSavedStateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "discard-saved-state (VM)",
		Short:   "Discard the saved state of a hibernated virtual machine, or cancel a hibernation that did not start saving yet, which keeps the virtual machine running. The next start boots it.",
		Example: usage(COMMAND_DISCARD_SAVED_STATE),
		Args:    cobra.ExactArgs(1),
		RunE:    discardSavedStateRun,
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func hibernateRun(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	options := &v1.HibernateOptions{DryRun: setDryRunOption(dryRun)}
	if storageClass != "" {
		options.StorageClassName = &storageClass
	}
	if err := virtClient.VirtualMachine(namespace).Hibernate(cmd.Context(), vmName, options); err != nil {
		return fmt.Errorf("error hibernating VirtualMachine %s: %v", vmName, err)
	}
	if !waitForSave || dryRun {
		cmd.Printf("VM %s was scheduled to hibernate\n", vmName)
		return nil
	}

	state, err := waitForSavedState(cmd.Context(), virtClient, namespace, vmName, waitTimeout)
	if err != nil {
		return err
	}
	if state.Phase == v1.HibernationFailed {
		return fmt.Errorf("hibernating VirtualMachine %s failed: %s", vmName, state.Message)
	}
	cmd.Printf("VM %s hibernated to PVC %s%s\n", vmName, state.ClaimName, saveDuration(state))
	return nil
}

// waitForSavedState waits until the hibernation of the VM ends in Failed, or
// in Saved and the VMI the state was saved from is gone. Its launcher pod
// takes a few seconds to terminate after the save, and a start issued before
// waits for it.
func waitForSavedState(ctx context.Context, virtClient kubecli.KubevirtClient, namespace, vmName string, timeout time.Duration) (*v1.VirtualMachineSavedState, error) {
	var state *v1.VirtualMachineSavedState
	err := wait.PollUntilContextTimeout(ctx, hibernatePollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		vm, err := virtClient.VirtualMachine(namespace).Get(ctx, vmName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		state = vm.Status.SavedState
		if state == nil {
			return false, nil
		}
		switch state.Phase {
		case v1.HibernationFailed:
			return true, nil
		case v1.HibernationSaved:
			return sourceGone(ctx, virtClient, namespace, vmName, state)
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error waiting for VirtualMachine %s to hibernate: %v", vmName, err)
	}
	return state, nil
}

// sourceGone reports whether the VMI the state was saved from is gone.
func sourceGone(ctx context.Context, virtClient kubecli.KubevirtClient, namespace, vmName string, state *v1.VirtualMachineSavedState) (bool, error) {
	vmi, err := virtClient.VirtualMachineInstance(namespace).Get(ctx, vmName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return vmi.UID != state.SourceUID, nil
}

func saveDuration(state *v1.VirtualMachineSavedState) string {
	if state.SaveStartTimestamp == nil || state.SaveEndTimestamp == nil {
		return ""
	}
	return fmt.Sprintf(", saving took %s", state.SaveEndTimestamp.Sub(state.SaveStartTimestamp.Time))
}

func discardSavedStateRun(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	if err := virtClient.VirtualMachine(namespace).DiscardSavedState(cmd.Context(), vmName); err != nil {
		return fmt.Errorf("error discarding the saved state of VirtualMachine %s: %v", vmName, err)
	}
	cmd.Printf("The saved state of VM %s was scheduled to be discarded\n", vmName)
	return nil
}
