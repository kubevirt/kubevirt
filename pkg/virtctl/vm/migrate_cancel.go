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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vm

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

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
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func migrateCancelRun(cmd *cobra.Command, args []string) error {
	vmiName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	// get a list of migrations for vmiName (use LabelSelector filter)
	labelselector := fmt.Sprintf("%s==%s", v1.MigrationSelectorLabel, vmiName)
	migrations, err := virtClient.VirtualMachineInstanceMigration(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelselector,
	})
	if err != nil {
		return fmt.Errorf("Error fetching virtual machine instance migration list  %v", err)
	}

	// There may be a single active migrations but several completed/failed ones
	// go over the migrations list and find the active one
	done := false
	for _, mig := range migrations.Items {
		migname := mig.ObjectMeta.Name

		if !mig.IsFinal() {
			// Cancel the active migration by calling Delete
			err = virtClient.VirtualMachineInstanceMigration(namespace).Delete(context.Background(), migname, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("Error canceling migration %s of a VirtualMachine %s: %v", migname, vmiName, err)
			}
			done = true
			break
		}
	}

	if !done {
		return fmt.Errorf("Found no migration to cancel for %s", vmiName)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, COMMAND_MIGRATE_CANCEL)

	return nil
}
