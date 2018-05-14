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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package wssh

import (
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"os"

	"k8s.io/client-go/tools/remotecommand"

	"strings"

	"strconv"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wssh (vm)",
		Short:   "Connect via websocket to ssh to a virtual machine, proxied via websockets.",
		Example: usage(),
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Console{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().BoolP("stdin", "i", false, "Pass stdin to the container")
	cmd.Flags().BoolP("tty", "t", false, "Stdin is a TTY")
	return cmd
}

type Console struct {
	clientConfig clientcmd.ClientConfig
}

func usage() string {
	usage := "# Connect via websocket to ssh to the virtual machine myvm:\n"
	usage += "virtctl wssh myvm"
	return usage
}

func (c *Console) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	cfg, err := c.clientConfig.ClientConfig()
	if err != nil {
		return err
	}
	req, err := kubecli.RequestFromConfig(cfg, args[0], namespace, "wssh")
	if err != nil {
		return err
	}

	stdin, err := cmd.Flags().GetBool("stdin")
	if err != nil {
		return err
	}

	tty, err := cmd.Flags().GetBool("tty")
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Set("tty", strconv.FormatBool(tty))
	q.Set("stdin", strconv.FormatBool(stdin))
	q.Set("stdout", "true")
	q.Set("stderr", strconv.FormatBool(!tty))
	q.Set("command", strings.Join(args[1:], " "))
	req.URL.RawQuery = q.Encode()
	req.URL.Scheme = "https"
	executor, err := remotecommand.NewSPDYExecutor(cfg, "GET", req.URL)
	if err != nil {
		return err
	}

	stderrStream := os.Stderr
	if tty {
		stderrStream = nil
	}

	return executor.Stream(remotecommand.StreamOptions{
		Tty:    tty,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: stderrStream,
	})
}
