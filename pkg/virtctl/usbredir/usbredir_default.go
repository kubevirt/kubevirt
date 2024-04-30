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
 * Copyright 2017, 2021 Red Hat, Inc.
 *
 */

package usbredir

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

//go:embed hwdata-usb.ids
var hwdata string

func (usbredirCmd *usbredirCommand) Run(command *cobra.Command, args []string) error {
	if _, err := exec.LookPath(usbredirClient); err != nil {
		return fmt.Errorf("Error on finding %s in $PATH: %s", usbredirClient, err.Error())
	}

	namespace, _, err := usbredirCmd.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(usbredirCmd.clientConfig)
	if err != nil {
		return err
	}

	vmiArg := args[1]
	usbdeviceArg := args[0]
	vendor, product, err := getDeviceMetadata(usbdeviceArg)
	if err != nil {
		log.Log.Reason(err).Info("Failed to find vendor & product info")
	}

	// Get connection to the websocket for usbredir subresource
	usbredirVMI, err := virtCli.VirtualMachineInstance(namespace).USBRedir(vmiArg, vendor, product)
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

	if usbredirClient, err := NewUSBRedirClient(ctx, "localhost:0", usbredirVMI); err != nil {
		return fmt.Errorf("Can't create usbredir client: %s", err.Error())
	} else {
		return usbredirClient.Redirect(usbdeviceArg)
	}
}

func getDeviceMetadata(arg string) (string, string, error) {
	var vendorHex, productHex string

	if strings.Contains(arg, ":") {
		sep := strings.Index(arg, ":")
		vendorHex, productHex = arg[:sep], arg[sep+1:]
	} else if strings.Contains(arg, "-") {
		return "", "", fmt.Errorf("Unsupported")
	}

	vendorInfo, productInfo, _ := MetadataLookup(hwdata, vendorHex, productHex)
	vendor := fmt.Sprintf("0x%s: %s", vendorHex, vendorInfo)
	product := fmt.Sprintf("0x%s: %s", productHex, productInfo)
	return vendor, product, nil
}
