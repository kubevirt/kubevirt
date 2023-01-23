package main

import (
	"path/filepath"
	"time"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

const (
	SocketDir  = util.VirtShareDir + "/sockets"
	SocketName = cmdclient.DedicatedCpuContainerSocketName
)

func main() {
	socketPath := filepath.Join(SocketDir, SocketName)

	_, err := grpcutil.CreateSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Error("Failed to start virt-launcher cmd server")
		panic(err)
	}

	for true {
		time.Sleep(123 * time.Hour)
	}
}
