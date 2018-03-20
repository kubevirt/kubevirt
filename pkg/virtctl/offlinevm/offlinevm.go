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

package offlinevm

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl"
)

const (
	COMMAND_START = "start"
	COMMAND_STOP  = "stop"
)

type Command struct {
	command string
}

func (c *Command) FlagSet() *flag.FlagSet {
	cf := flag.NewFlagSet(c.command, flag.ExitOnError)

	return cf
}

func NewCommand(command string) *Command {
	return &Command{command: command}
}

func (c *Command) Usage() string {
	virtctlCmd := path.Base(os.Args[0])
	usage := fmt.Sprintf("%s an OfflineVirtualMachine\n\n", strings.Title(c.command))
	usage += "Example:\n"
	usage += fmt.Sprintf("%s %s myvm\n", virtctlCmd, c.command)
	return usage
}

func (o *Command) Run(flags *flag.FlagSet) int {
	var virtClient kubecli.KubevirtClient
	var err error

	if flags.NArg() != 2 {
		log.Println("OfflineVirtualMachine name is missing")
		return virtctl.STATUS_ERROR
	}
	vmName := flags.Arg(1)

	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	namespace, _ := flags.GetString("namespace")

	var running bool
	command := flags.Arg(0)
	if command == COMMAND_START {
		running = true
	} else if command == COMMAND_STOP {
		running = false
	}

	if (server != "") && (kubeconfig != "") {
		virtClient, err = kubecli.GetKubevirtClientFromFlags(server, kubeconfig)
	} else {
		virtClient, err = kubecli.GetKubevirtClient()
	}
	if err != nil {
		log.Printf("Cannot obtain KubeVirt client: %v", err)
		return virtctl.STATUS_ERROR
	}

	options := &k8smetav1.GetOptions{}
	ovm, err := virtClient.OfflineVirtualMachine(namespace).Get(vmName, options)
	if err != nil {
		log.Printf("Error fetching OfflineVirtualMachine: %v", err)
		return virtctl.STATUS_ERROR
	}

	if ovm.Spec.Running != running {
		ovm.Spec.Running = running
		_, err := virtClient.OfflineVirtualMachine(namespace).Update(ovm)
		if err != nil {
			log.Printf("Error updating OfflineVirtualMachine: %v", err)
			return virtctl.STATUS_ERROR
		}
	} else {
		stateMsg := "stopped"
		if running {
			stateMsg = "running"
		}
		log.Printf("Error: VirtualMachine '%s' is already %s", vmName, stateMsg)
		return virtctl.STATUS_ERROR
	}

	return virtctl.STATUS_OK
}
