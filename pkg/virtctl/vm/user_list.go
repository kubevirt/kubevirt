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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

const COMMAND_USERLIST = "userlist"

func NewUserListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "userlist (VMI)",
		Short:   "Return full list of logged in users on the guest machine.",
		Example: usage(COMMAND_USERLIST),
		Args:    cobra.ExactArgs(1),
		RunE:    userListRun,
	}
	return cmd
}

func userListRun(cmd *cobra.Command, args []string) error {
	vmiName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	userlist, err := virtClient.VirtualMachineInstance(namespace).UserList(context.Background(), vmiName)
	if err != nil {
		return fmt.Errorf("Error listing users of VirtualMachineInstance %s, %v", vmiName, err)
	}

	data, err := json.MarshalIndent(userlist, "", "  ")
	if err != nil {
		return fmt.Errorf("Cannot marshal userlist %v", err)
	}

	fmt.Printf("%s\n", string(data))
	return nil
}
