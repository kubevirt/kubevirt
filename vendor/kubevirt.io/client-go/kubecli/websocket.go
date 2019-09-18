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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package kubecli

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/gorilla/websocket"

	"kubevirt.io/client-go/subresources"
)

const (
	WebsocketMessageBufferSize = 10240
	wsFrameHeaderSize          = 2 + 8 + 4 // Fixed header + length + mask (RFC 6455)
)

func NewUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  WebsocketMessageBufferSize,
		WriteBufferSize: WebsocketMessageBufferSize,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
		Subprotocols: []string{subresources.PlainStreamProtocolName},
	}
}

func Dial(address string, tlsConfig *tls.Config) (*websocket.Conn, *http.Response, error) {
	dialer := &websocket.Dialer{
		ReadBufferSize:  WebsocketMessageBufferSize,
		WriteBufferSize: WebsocketMessageBufferSize,
		Subprotocols:    []string{subresources.PlainStreamProtocolName},
		TLSClientConfig: tlsConfig,
	}
	return dialer.Dial(address, nil)
}

func Copy(dst *websocket.Conn, src *websocket.Conn) (written int64, err error) {
	return copy(&binaryWriter{conn: dst}, &binaryReader{conn: src})
}

func CopyFrom(dst io.Writer, src *websocket.Conn) (written int64, err error) {
	return copy(dst, &binaryReader{conn: src})
}

func CopyTo(dst *websocket.Conn, src io.Reader) (written int64, err error) {
	return copy(&binaryWriter{conn: dst}, src)
}

func copy(dst io.Writer, src io.Reader) (written int64, err error) {
	// our websocket package has an issue where it truncates messages
	// when the message+header is greater than the buffer size we allocate.
	// thus, we copy in chunks of WebsocketMessageBufferSize-wsFrameHeaderSize
	buf := make([]byte, WebsocketMessageBufferSize-wsFrameHeaderSize)
	return io.CopyBuffer(dst, src, buf)
}

type binaryWriter struct {
	conn *websocket.Conn
}

func (s *binaryWriter) Write(p []byte) (int, error) {
	w, err := s.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, convert(err)
	}
	defer w.Close()
	return w.Write(p)
}

type binaryReader struct {
	conn *websocket.Conn
}

func (s *binaryReader) Read(p []byte) (int, error) {
	for {
		msgType, r, err := s.conn.NextReader()
		if err != nil {
			return 0, convert(err)
		}

		switch msgType {
		case websocket.BinaryMessage:
			n, err := r.Read(p)
			return n, convert(err)

		case websocket.CloseMessage:
			return 0, io.EOF
		}
	}
}

func convert(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
		return io.EOF
	}
	return err
}
