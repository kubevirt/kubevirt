package rest

import (
	"context"
	"io"
	"net"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

type vmiFetcher func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError)
type dialer func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError)
type validator func(vmi *v1.VirtualMachineInstance) *errors.StatusError
type streamFunc func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult)
type streamFuncResult error

type Streamer struct {
	fetchVMI        vmiFetcher
	validateVMI     validator
	dial            dialer
	keepAliveClient func(ctx context.Context, conn *websocket.Conn, cancel func())

	streamToClient streamFunc
	streamToServer streamFunc
}

func NewRawStreamer(fetch vmiFetcher, validate validator, dial dialer) *Streamer {
	return &Streamer{
		fetchVMI:    fetch,
		validateVMI: validate,
		dial:        dial,
		streamToServer: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			_, err := io.Copy(serverConn, clientConn.UnderlyingConn())
			result <- err
		},
		streamToClient: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			_, err := io.Copy(clientConn.UnderlyingConn(), serverConn)
			result <- err
		},
	}
}

func NewWebsocketStreamer(fetch vmiFetcher, validate validator, dial dialer) *Streamer {
	return &Streamer{
		fetchVMI:        fetch,
		validateVMI:     validate,
		dial:            dial,
		keepAliveClient: keepAliveClientStream,
		streamToServer: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			_, err := kubecli.CopyFrom(serverConn, clientConn)
			result <- err
		},
		streamToClient: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			_, err := kubecli.CopyTo(clientConn, serverConn)
			result <- err
		},
	}
}

func (s *Streamer) Handle(request *restful.Request, response *restful.Response) error {
	namespace := request.PathParameter(NamespaceParamName)
	name := request.PathParameter(NameParamName)

	vmi, statusErr := s.fetchAndValidateVMI(namespace, name)
	if statusErr != nil {
		writeError(statusErr, response)
		return statusErr
	}

	serverConn, statusErr := s.dial(vmi)
	if statusErr != nil {
		writeError(statusErr, response)
		return statusErr
	}
	clientConn, err := clientConnectionUpgrade(request, response)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return err
	}

	ctx, cancel := context.WithCancel(request.Request.Context())
	defer cancel()
	go s.cleanupOnClosedContext(ctx, clientConn, serverConn)

	if s.keepAliveClient != nil {
		go s.keepAliveClient(context.Background(), clientConn, cancel)
	}

	results := make(chan streamFuncResult, 2)
	defer close(results)

	go s.streamToClient(clientConn, serverConn, results)
	go s.streamToServer(clientConn, serverConn, results)

	result1 := <-results
	// start canceling on the first result to force all goroutines to terminate
	cancel()
	result2 := <-results

	if result1 != nil {
		return result1
	}
	return result2
}

func (s *Streamer) fetchAndValidateVMI(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := s.fetchVMI(namespace, name)
	if err != nil {
		return nil, err
	}
	if err := s.validateVMI(vmi); err != nil {
		return nil, err
	}
	return vmi, nil
}

const streamTimeout = 10 * time.Second

func clientConnectionUpgrade(request *restful.Request, response *restful.Response) (*websocket.Conn, error) {
	upgrader := kubecli.NewUpgrader()
	upgrader.HandshakeTimeout = streamTimeout
	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		return nil, err
	}
	return clientSocket, nil
}

func (s *Streamer) cleanupOnClosedContext(ctx context.Context, clientConn *websocket.Conn, serverConn net.Conn) {
	<-ctx.Done()
	serverConn.Close()
	clientConn.Close()
}

const keepAliveTimeout = 1 * time.Minute

func keepAliveClientStream(ctx context.Context, conn *websocket.Conn, cancel func()) {
	pingTicker := time.NewTicker(1 * time.Second)
	defer pingTicker.Stop()
	conn.SetReadDeadline(time.Now().Add(keepAliveTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(keepAliveTimeout))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("keep alive"), time.Now().Add(keepAliveTimeout)); err != nil {
				log.Log.Reason(err).Error("Failed to write control message to client websocket connection")
				cancel()
				return
			}
		}
	}
}
