package launcher

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"
)

type registerCallbackServersFn func(*grpc.Server, chan struct{})

func Run(socketPath string, registerCallbackServers registerCallbackServersFn) error {
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("Failed to initialized socket on path %q: %v", socket, err)
	}
	defer func() {
		os.Remove(socketPath)
	}()

	shutdownChan := make(chan struct{})
	server := grpc.NewServer([]grpc.ServerOption{}...)
	registerCallbackServers(server, shutdownChan)

	// Handle signals to properly shutdown process
	signalStopChan := make(chan os.Signal, 1)
	signal.Notify(signalStopChan, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	log.Log.Infof("shim is now exposing its services on socket %s", socketPath)
	errChan := make(chan error)
	go func() {
		errChan <- server.Serve(socket)
	}()

	select {
	case s := <-signalStopChan:
		log.Log.Infof("sidecar-shim received signal: %s", s.String())
	case err = <-errChan:
		log.Log.Reason(err).Error("Failed to run grpc server")
	case <-shutdownChan:
		log.Log.Info("Exiting")
	}

	if err == nil {
		server.GracefulStop()
	}

	return nil
}
