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
 * Copyright The KubeVirt Authors
 *
 */

package v1

import (
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type wsStreamer struct {
	conn *websocket.Conn
	done chan struct{}
}

func (ws *wsStreamer) streamDone() {
	close(ws.done)
}

func (ws *wsStreamer) Stream(options StreamOptions) error {
	copyErr := make(chan error, 1)

	go func() {
		_, err := CopyTo(ws.conn, options.In)
		copyErr <- err
	}()

	go func() {
		_, err := CopyFrom(options.Out, ws.conn)
		copyErr <- err
	}()

	defer ws.streamDone()
	return <-copyErr
}

func (ws *wsStreamer) AsConn() net.Conn {
	return &wsConn{
		Conn:         ws.conn,
		binaryReader: &binaryReader{conn: ws.conn},
		binaryWriter: &binaryWriter{conn: ws.conn},
	}
}

type wsConn struct {
	*websocket.Conn
	*binaryReader
	*binaryWriter
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.Conn.SetWriteDeadline(t); err != nil {
		return err
	}
	return c.Conn.SetReadDeadline(t)
}

func NewWebsocketStreamer(conn *websocket.Conn, done chan struct{}) *wsStreamer {
	return &wsStreamer{
		conn: conn,
		done: done,
	}
}
