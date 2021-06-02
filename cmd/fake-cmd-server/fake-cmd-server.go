package main

import (
	"errors"
	goflag "flag"

	"github.com/golang/mock/gomock"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
)

func main() {
	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")

	socket := pflag.String("socket", cmdclient.SocketOnGuest(), "Socket for the cmd server")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.Parse()

	log.InitializeLogging("fake-cmd-server")

	stopChan := make(chan struct{})
	options := cmdserver.NewServerOptions(true)

	domainManager := virtwrap.NewMockDomainManager(gomock.NewController(nil))
	domainManager.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().DoAndReturn(func(domainName string, _ string, _ []string) (string, error) {
		if domainName == "error" {
			return "", errors.New("fake error")
		}
		if domainName == "fail" {
			return "command failed", agent.ExecExitCode{ExitCode: 1}
		}
		return "success", nil
	})
	log.Log.Info("running fake server")
	done, err := cmdserver.RunServer(*socket, domainManager, stopChan, options)
	if err != nil {
		log.Log.Reason(err).Critical("running cmd server")
	}

	<-done
}
