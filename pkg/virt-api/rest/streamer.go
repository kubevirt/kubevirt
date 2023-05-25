package rest

import (
	"context"
	"io"
	"net"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type vmiFetcher func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError)
type validator func(vmi *v1.VirtualMachineInstance) *errors.StatusError
type streamFunc func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult)
type streamFuncResult error

type dialer interface {
	Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *errors.StatusError)
	DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError)
}

type Streamer struct {
	dialer          *DirectDialer
	keepAliveClient func(ctx context.Context, conn *websocket.Conn, cancel func())

	streamToClient streamFunc
	streamToServer streamFunc
}

type DirectDialer struct {
	fetchVMI    vmiFetcher
	validateVMI validator
	dial        dialer
}

func NewRawStreamer(fetch vmiFetcher, validate validator, dial dialer) *Streamer {
	return &Streamer{
		dialer: NewDirectDialer(fetch, validate, dial),
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
		dialer:          NewDirectDialer(fetch, validate, dial),
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
	namespace := request.PathParameter(definitions.NamespaceParamName)
	name := request.PathParameter(definitions.NameParamName)
	serverConn, statusErr := s.dialer.DialUnderlying(namespace, name)

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

func NewDirectDialer(fetch vmiFetcher, validate validator, dial dialer) *DirectDialer {
	return &DirectDialer{
		fetchVMI:    fetch,
		validateVMI: validate,
		dial:        dial,
	}
}

func (d *DirectDialer) Dial(namespace, name string) (*websocket.Conn, *errors.StatusError) {
	vmi, err := d.fetchAndValidateVMI(namespace, name)
	if err != nil {
		return nil, err
	}

	return d.dial.Dial(vmi)
}

func (d *DirectDialer) DialUnderlying(namespace, name string) (net.Conn, *errors.StatusError) {
	vmi, err := d.fetchAndValidateVMI(namespace, name)
	if err != nil {
		return nil, err
	}

	return d.dial.DialUnderlying(vmi)
}

func (d *DirectDialer) fetchAndValidateVMI(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := d.fetchVMI(namespace, name)
	if err != nil {
		return nil, err
	}
	if err := d.validateVMI(vmi); err != nil {
		return nil, err
	}
	return vmi, nil
}
