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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_USERLIST = "userlist"

	volumeNameArg         = "volume-name"
	notDefinedGracePeriod = -1
	dryRunCommandUsage    = "--dry-run=false: Flag used to set whether to perform a dry run or not. If true the command will be executed without performing any changes."

	dryRunArg      = "dry-run"
	forceArg       = "force"
	gracePeriodArg = "grace-period"
	persistArg     = "persist"

	YAML = "yaml"
	JSON = "json"
)

var (
	vmiName      string
	forceRestart bool
	gracePeriod  int64
	volumeName   string
	persist      bool
	dryRun       bool
)

func NewUserListCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "userlist (VMI)",
		Short:   "Return full list of logged in users on the guest machine.",
		Example: usage(COMMAND_USERLIST),
		Args:    templates.ExactArgs("userlist", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_USERLIST, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
	cmd          *cobra.Command
}

func usage(cmd string) string {
	if cmd == COMMAND_USERLIST || cmd == COMMAND_FSLIST || cmd == COMMAND_GUESTOSINFO {
		usage := fmt.Sprintf("  # %s a virtual machine instance called 'myvm':\n", strings.Title(cmd))
		usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
		return usage
	}

	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
	return usage
}

func gracePeriodIsSet(period int64) bool {
	return period != notDefinedGracePeriod
}

func getDryRunOption(dryRun bool) []string {
	if dryRun {
		return []string{metav1.DryRunAll}
	}
	return nil
}

func (o *Command) Run(args []string) error {
	if len(args) != 0 {
		vmiName = args[0]
	}

	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	dryRunOption := getDryRunOption(dryRun)
	if len(dryRunOption) > 0 && dryRunOption[0] == metav1.DryRunAll {
		fmt.Printf("Dry Run execution\n")
	}
	switch o.command {
	case COMMAND_USERLIST:
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

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
