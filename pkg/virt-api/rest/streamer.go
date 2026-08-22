/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package rest

import (
	"context"
	"io"
	"net"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"

	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type streamFunc func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- error)

type Streamer struct {
	dialer          *DirectDialer
	keepAliveClient func(ctx context.Context, conn *websocket.Conn, cancel func())

	streamToClient streamFunc
	streamToServer streamFunc
}

func NewRawStreamer(dialer *DirectDialer) *Streamer {
	return &Streamer{
		dialer: dialer,
		streamToServer: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- error) {
			_, err := io.Copy(serverConn, clientConn.UnderlyingConn())
			result <- err
		},
		streamToClient: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- error) {
			_, err := io.Copy(clientConn.UnderlyingConn(), serverConn)
			result <- err
		},
	}
}

func NewWebsocketStreamer(dialer *DirectDialer) *Streamer {
	return &Streamer{
		dialer:          dialer,
		keepAliveClient: keepAliveClientStream,
		streamToServer: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- error) {
			_, err := kvcorev1.CopyFrom(serverConn, clientConn)
			result <- err
		},
		streamToClient: func(clientConn *websocket.Conn, serverConn net.Conn, result chan<- error) {
			_, err := kvcorev1.CopyTo(clientConn, serverConn)
			result <- err
		},
	}
}

func (s *Streamer) Handle(request *restful.Request, response *restful.Response) error {
	namespace := request.PathParameter(definitions.NamespaceParamName)
	name := request.PathParameter(definitions.NameParamName)
	serverConn, statusErr := s.dialer.Dial(namespace, name)

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

	results := make(chan error, 2)
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
	upgrader := kvcorev1.NewUpgrader()
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
