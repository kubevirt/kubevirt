package grpc

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

type registerFunc func(grpcServer *grpc.Server)

func RunServer(stop <-chan struct{}, socketPath string, registerF ...registerFunc) (<-chan struct{}, error) {
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)

	for _, f := range registerF {
		f(grpcServer)
	}

	sock, err := grpcutil.CreateSocket(socketPath)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		<-stop
		log.Log.Info("stopping grpc server")
		stopped := make(chan struct{})
		go func() {
			grpcServer.Stop()
			close(stopped)
		}()

		select {
		case <-stopped:
			log.Log.Info("grpc server stopped")
		case <-time.After(1 * time.Second):
			log.Log.Error("timeout on stopping the grpc server, continuing anyway.")
		}
		sock.Close()
		os.Remove(socketPath)
		close(done)
	}()

	go func() {
		grpcServer.Serve(sock)
	}()

	return done, nil
}
