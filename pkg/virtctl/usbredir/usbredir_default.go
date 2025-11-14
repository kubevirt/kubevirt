//go:build !s390x

/* Licensed under the Apache License, Version 2.0 (the "License");
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

package usbredir

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

func Run(cmd *cobra.Command, args []string) error {
	disableLaunch := cmd.Flags().Changed(optDisableClientLaunch)
	if !disableLaunch && len(args) != 2 {
		return fmt.Errorf("Missing argument")
	}

	if _, err := exec.LookPath(usbredirClient); err != nil && !disableLaunch {
		return fmt.Errorf("Error on finding %s in $PATH: %w", usbredirClient, err)
	}

	var vmiArg, usbdeviceArg string
	if disableLaunch {
		vmiArg = args[0]
	} else {
		usbdeviceArg = args[0]
		vmiArg = args[1]
	}

	virtCli, _, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	// Get connection to the websocket for usbredir subresource
	usbredirVMI, err := virtCli.VirtualMachineInstance(namespace).USBRedir(vmiArg)
	if err != nil {
		return fmt.Errorf("Can't access VMI %s: %s", vmiArg, err.Error())
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	go func(cancelFn context.CancelFunc) {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		select {
		case <-interrupt:
			cancelFn()
		case <-ctx.Done():
			signal.Stop(interrupt)
		}
	}(cancelFn)

	usbredirClient, err := NewUSBRedirClient(ctx, "localhost:0", usbredirVMI)
	if err != nil {
		return fmt.Errorf("Can't create usbredir client: %s", err.Error())
	}

	if disableLaunch {
		// This is a log to the user, should use stdout instead of default stderr of log
		cmd.Printf("User can connect usbredir client at: %s\n", usbredirClient.GetProxyAddress())
		usbredirClient.LaunchClient = false
	}
	return usbredirClient.Redirect(usbdeviceArg)
}
