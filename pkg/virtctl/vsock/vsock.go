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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	tlsFlag  = "tls"
	argCount = 2
	logLevel = 3
)

type vsockCmd struct {
	useTLS bool
}

func NewCommand() *cobra.Command {
	log.InitializeLogging("vsock")
	c := &vsockCmd{useTLS: true}
	cmd := &cobra.Command{
		Use:   "vsock (VM|VMI) PORT",
		Short: "Open a vsock connection to a virtual machine instance.",
		Long: "Open a vsock connection to a virtual machine instance.\n\n" +
			"PORT is a TCP/UDP-style application port (1-65535) that the application inside the guest " +
			"is listening on over VSOCK, not the raw 32-bit VSOCK port field.",
		Example: usage(),
		Args:    cobra.ExactArgs(argCount),
		RunE:    c.run,
	}
	cmd.Flags().BoolVar(&c.useTLS, tlsFlag, c.useTLS,
		fmt.Sprintf("--%s=true: Use TLS for the vsock connection (requires the application listening on the VSOCK port to support TLS)", tlsFlag))
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

	_, ns, name, err := portforward.ParseTarget(args[0])
	if err != nil {
		return err
	}
	if ns != "" {
		namespace = ns
	}

	// VSOCK requires a running VMI; in KubeVirt the VMI always shares the name with its VM,
	// so this lookup applies the same way whether the target was given as vm/ or vmi/.
	vmiClient := client.VirtualMachineInstance(namespace)
	vmi, err := vmiClient.Get(cmd.Context(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to find VirtualMachineInstance %q: %w", name, err)
	}
	if vmi.Status.Phase != v1.Running {
		return fmt.Errorf("VirtualMachineInstance %q is not running (phase: %s)", name, vmi.Status.Phase)
	}

	port, err := strconv.ParseUint(args[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", args[1], err)
	}
	if port == 0 {
		return fmt.Errorf("invalid port %q: port must be greater than 0", args[1])
	}

	streamer, err := vmiClient.VSOCK(name, &v1.VSOCKOptions{
		TargetPort: uint32(port),
		UseTLS:     &c.useTLS,
	})
	if err != nil {
		return err
	}

	log.Log.V(logLevel).Infof("vsock stream to %s/%s port %d", namespace, name, port)
	return streamer.Stream(kvcorev1.StreamOptions{
		In:  os.Stdin,
		Out: os.Stdout,
	})
}

func usage() string {
	return `  # Open a vsock connection to 'testvmi' on port 22:
  {{ProgramName}} vsock vmi/testvmi 22

  # Open a vsock connection to 'testvmi' in 'mynamespace' on port 22:
  {{ProgramName}} vsock vmi/testvmi/mynamespace 22

  # Open a vsock connection to the running VirtualMachineInstance of 'testvm' on port 22:
  {{ProgramName}} vsock vm/testvm 22`
}
