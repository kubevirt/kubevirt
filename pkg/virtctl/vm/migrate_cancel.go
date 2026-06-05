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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_MIGRATE_CANCEL = "migrate-cancel"

func NewMigrateCancelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate-cancel (VM)",
		Short:   "Cancel migration of a virtual machine.",
		Example: usage(COMMAND_MIGRATE_CANCEL),
		Args:    cobra.ExactArgs(1),
		RunE:    migrateCancelRun,
	}

	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func migrateCancelRun(cmd *cobra.Command, args []string) error {
	vmiName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	err = migrateCancel(cmd.Context(), virtClient, vmiName, namespace)
	if err != nil {
		return err
	}

	cmd.Printf("VM %s was scheduled to %s\n", vmiName, COMMAND_MIGRATE_CANCEL)
	return nil
}

func migrateCancel(ctx context.Context, virtClient kubecli.KubevirtClient, vmiName string, namespace string) error {
	// get a list of migrations for vmiName (use LabelSelector filter)
	labelSelector := fmt.Sprintf("%s==%s", v1.MigrationSelectorLabel, vmiName)
	migrations, err := virtClient.VirtualMachineInstanceMigration(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("Error fetching virtual machine instance migration list  %v", err)
	}

	deleteOpts := metav1.DeleteOptions{
		DryRun: setDryRunOption(dryRun),
	}

	// There may be a single active migrations but several completed/failed ones
	// go over the migrations list and find the active one
	for _, mig := range migrations.Items {
		if mig.IsFinal() {
			continue
		}

		migName := mig.ObjectMeta.Name

		// Cancel the active migration by calling Delete
		err = virtClient.VirtualMachineInstanceMigration(namespace).Delete(ctx, migName, deleteOpts)
		if err != nil {
			return fmt.Errorf("Error canceling migration %s of a VirtualMachine %s: %v", migName, vmiName, err)
		}

		return nil
	}

	return fmt.Errorf("Found no migration to cancel for %s", vmiName)
}
