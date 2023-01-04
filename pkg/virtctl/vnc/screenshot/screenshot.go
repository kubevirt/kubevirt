package screenshot

import (
	"context"
	"fmt"
	"os"

	v1 "kubevirt.io/api/core/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewScreenshotCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	s := Screenshot{clientConfig: clientConfig}
	cmd := &cobra.Command{
		Use:     "screenshot (VMI)",
		Short:   "Create a VNC screenshot of a virtual machine instance.",
		Example: usage(),
		Args:    templates.ExactArgs("screenshot", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := s
			return c.Run(cmd, args)
		},
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
	clientConfig clientcmd.ClientConfig
	fileName     string
	moveCursor   bool
}

func (s *Screenshot) Run(_ *cobra.Command, args []string) error {
	namespace, _, err := s.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(s.clientConfig)
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
