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

package v1

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/gorilla/websocket"

	"kubevirt.io/client-go/subresources"
)

const (
	WebsocketMessageBufferSize = 10240
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

func Copy(dst *websocket.Conn, src *websocket.Conn) (int64, error) {
	return io.Copy(dst.UnderlyingConn(), src.UnderlyingConn())
}

func CopyFrom(dst io.Writer, src *websocket.Conn) (written int64, err error) {
	return io.Copy(dst, &binaryReader{conn: src})
}

func CopyTo(dst *websocket.Conn, src io.Reader) (written int64, err error) {
	return io.Copy(&binaryWriter{conn: dst}, src)
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
	n, err := w.Write(p)
	return n, err
}

type binaryReader struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (s *binaryReader) Read(p []byte) (int, error) {
	var msgType int
	var err error
	for {
		if s.reader == nil {
			msgType, s.reader, err = s.conn.NextReader()
		} else {
			msgType = websocket.BinaryMessage
		}
		if err != nil {
			s.reader = nil
			return 0, convert(err)
		}

		switch msgType {
		case websocket.BinaryMessage:
			n, readErr := s.reader.Read(p)
			err = readErr
			if err != nil {
				s.reader = nil
				if err == io.EOF {
					if n == 0 {
						continue
					} else {
						return n, nil
					}
				}
			}
			return n, convert(err)
		case websocket.CloseMessage:
			return 0, io.EOF
		default:
			s.reader = nil
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
