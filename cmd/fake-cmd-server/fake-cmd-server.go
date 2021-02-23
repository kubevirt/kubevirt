package main

import (
	goflag "flag"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
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

	log.Log.Info("running fake server")
	done, err := cmdserver.RunServer(*socket, FakeDomainManager{}, stopChan, options)
	if err != nil {
		log.Log.Reason(err).Critical("running cmd server")
	}

	<-done
}
