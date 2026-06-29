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

package vsock

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	tlsFlag = "tls"
)

type vsockCmd struct {
	useTLS bool
}

func NewCommand() *cobra.Command {
	log.InitializeLogging("vsock")
	c := &vsockCmd{useTLS: true}
	cmd := &cobra.Command{
		Use:     "vsock VMI PORT",
		Short:   "Open a vsock connection to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.run,
	}
	cmd.Flags().BoolVar(&c.useTLS, tlsFlag, c.useTLS,
		fmt.Sprintf("--%s=true: Use TLS for the vsock connection", tlsFlag))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (c *vsockCmd) run(cmd *cobra.Command, args []string) error {
	// Redirect output to stderr: stdout is used for the binary stream.
	cmd.SetOut(os.Stderr)
	cmd.Root().SetOut(os.Stderr)

	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	kind, ns, name, err := portforward.ParseTarget(args[0])
	if err != nil {
		return err
	}
	if kind != "vmi" {
		return fmt.Errorf("vsock is only supported for VirtualMachineInstances (vmi), got %q", kind)
	}
	if ns != "" {
		namespace = ns
	}

	port, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		return fmt.Errorf("invalid port %q: %v", args[1], err)
	}

	streamer, err := client.VirtualMachineInstance(namespace).VSOCK(name, &v1.VSOCKOptions{
		TargetPort: uint32(port),
		UseTLS:     &c.useTLS,
	})
	if err != nil {
		return err
	}

	log.Log.V(3).Infof("vsock stream to %s/%s port %d", namespace, name, port)
	return streamer.Stream(kvcorev1.StreamOptions{
		In:  os.Stdin,
		Out: os.Stdout,
	})
}

func usage() string {
	return `  # Open a vsock connection to 'testvmi' on port 22:
  {{ProgramName}} vsock vmi/testvmi 22

  # Open a vsock connection to 'testvmi' in 'mynamespace' on port 22:
  {{ProgramName}} vsock vmi/testvmi/mynamespace 22`
}
