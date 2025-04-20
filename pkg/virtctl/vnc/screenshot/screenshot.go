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
 */

package screenshot

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewScreenshotCommand() *cobra.Command {
	s := Screenshot{}
	cmd := &cobra.Command{
		Use:     "screenshot (VMI)",
		Short:   "Create a VNC screenshot of a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE:    s.Run,
	}
	cmd.Flags().StringVarP(&s.fileName, "file", "f", "", "where to store the VNC screenshot in PNG format. User '-' for stdout")
	cmd.Flags().BoolVarP(&s.moveCursor, "move-cursor", "m", false, "move the cursor to wake up the screen in case of screensavers")
	err := cmd.MarkFlagRequired("file")
	if err != nil {
		panic(err)
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	return `   # Take a VNC screenshot of 'testvmi' in png format:
   {{ProgramName}} vnc screenshot testvmi -f screenshot.png

   # Take a VNC screenshot of 'testvmi' in png format and pipe it through "display" to show it right away:
   {{ProgramName}} vnc screenshot testvmi -f - | display`
}

type Screenshot struct {
	fileName   string
	moveCursor bool
}

func (s *Screenshot) Run(cmd *cobra.Command, args []string) error {
	virtCli, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	// setup connection with VM
	vmi := args[0]
	screenshot, err := virtCli.VirtualMachineInstance(namespace).Screenshot(context.Background(), vmi, &v1.ScreenshotOptions{MoveCursor: s.moveCursor})
	if err != nil {
		return fmt.Errorf("Can't access VMI %s: %v", vmi, err)
	}

	if s.fileName == "-" {
		if _, err := os.Stdout.Write(screenshot); err != nil {
			return fmt.Errorf("failed to write image to stdout: %v", err)
		}
	} else if err := os.WriteFile(s.fileName, screenshot, 0644); err != nil {
		return fmt.Errorf("Can't write image to a file: %v", err)
	}
	return nil
}
