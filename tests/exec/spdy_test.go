/*
 * This file is part of the kubevirt project
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

package exec

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func TestCancellableSPDYStreamsOutput(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := httpstream.Handshake(r, w, []string{"v4.channel.k8s.io"}); err != nil {
			t.Error(err)
			return
		}
		type streamReply struct {
			stream httpstream.Stream
			reply  <-chan struct{}
		}
		streams := make(chan streamReply, 3)
		conn := spdy.NewResponseUpgrader().UpgradeResponse(w, r, func(s httpstream.Stream, reply <-chan struct{}) error {
			streams <- streamReply{s, reply}
			return nil
		})
		if conn == nil {
			return
		}
		defer conn.Close()
		var errorStream httpstream.Stream
		for range 3 {
			select {
			case received := <-streams:
				<-received.reply
				switch received.stream.Headers().Get(v1.StreamType) {
				case v1.StreamTypeStdout:
					if _, err := received.stream.Write([]byte("route output")); err != nil {
						t.Error(err)
					}
					received.stream.Close()
				case v1.StreamTypeStderr:
					received.stream.Close()
				case v1.StreamTypeError:
					errorStream = received.stream
				}
			case <-done:
				return
			}
		}
		if errorStream != nil {
			if _, err := errorStream.Write([]byte(`{"status":"Success"}`)); err != nil {
				t.Error(err)
			}
			errorStream.Close()
		}
		<-done
	}))
	defer server.Close()
	defer close(done)
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := newCancellableSPDYExecutor(&rest.Config{Host: server.URL, TLSClientConfig: rest.TLSClientConfig{Insecure: true}}, "POST", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "route output" || stderr.Len() != 0 {
		t.Fatalf("incorrect streams: stdout=%q stderr=%q", &stdout, &stderr)
	}
}

func TestCancellableSPDYRejectsMalformedUpgradeWithoutReadingStream(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		if _, err := buffer.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: another-protocol\r\n\r\n"); err != nil {
			t.Error(err)
			return
		}
		if err := buffer.Flush(); err != nil {
			t.Error(err)
			return
		}
		<-done // An invalid upgraded peer need never send response-body bytes.
	}))
	defer server.Close()
	defer close(done)
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := newCancellableSPDYExecutor(&rest.Config{Host: server.URL}, "POST", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- executor.StreamWithContext(ctx, remotecommand.StreamOptions{}) }()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("accepted the wrong upgrade protocol")
		}
	case <-time.After(time.Second):
		t.Fatal("read malformed upgraded stream past its deadline")
	}
}
