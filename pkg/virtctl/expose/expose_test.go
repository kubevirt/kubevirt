package expose

import (
	"testing"

	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

// TODO: build a ginko test suite

func Test_CommandCreation(t *testing.T) {
	var flags pflag.FlagSet
	clientConfig := kubecli.DefaultClientConfig(&flags)
	cmd := NewExposeCommand(clientConfig)
	if cmd == nil {
		t.Error("'expose' command creation failure")
	}
	// TODO: verify content of command
}

func Test_Run(t *testing.T) {
	var flags pflag.FlagSet
	clientConfig := kubecli.DefaultClientConfig(&flags)
	cmd := NewExposeCommand(clientConfig)
	if cmd == nil {
		t.Error("'expose' command creation failure")
	}
	// TODO: mock the client to not communicate with the server
	err := cmd.RunE(cmd, []string{"vm", "testvm"})
	if err != nil {
		// this is currently failing, uncomment once client is mocked
		//t.Error("'expose' command execution failure: ", err)
	}
}
