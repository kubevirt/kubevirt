package main

import (
	"path/filepath"
	"time"

	"github.com/spf13/pflag"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

func main() {
	virtShareDir := pflag.String("virt-share-dir", util.VirtShareDir, "kubevirt public shared dir")
	pflag.Parse()

	socketDir := filepath.Join(*virtShareDir, "sockets")

	if exists, err := ephemeraldiskutils.FileExists(socketDir); !exists || err != nil {
		// This directory will be created by virt-handler
		if err != nil {
			log.Log.Reason(err).Errorf("while while checking existance of %s directory", socketDir)
		}
		time.Sleep(2 * time.Second)
	}

	socketPath := filepath.Join(socketDir, cmdclient.EmulatorContainerSocketName)

	listener, err := grpcutil.CreateSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Error("Failed to start virt-launcher cmd server")
		panic(err)
	}

	log.Log.Infof("socket %s is created successfully. waiting for connections...", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Log.Reason(err).Errorf("couldn't accept connection")
			continue
		}
		log.Log.Infof("received new connection from %s", conn.RemoteAddr().String())

		err = conn.SetDeadline(time.Now().Add(5 * time.Minute))
		if err != nil {
			log.Log.Errorf("failed setting connection deadline: %v", err)
		}
	}
}
