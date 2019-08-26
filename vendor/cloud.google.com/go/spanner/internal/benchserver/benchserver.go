/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package benchserver

import (
	"context"
	"encoding/binary"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	proto3 "github.com/golang/protobuf/ptypes/struct"
	pbt "github.com/golang/protobuf/ptypes/timestamp"
	sppb "google.golang.org/genproto/googleapis/spanner/v1"
	"google.golang.org/grpc"
)

var (
	// KvMeta is the Metadata for mocked KV table.
	KvMeta = sppb.ResultSetMetadata{
		RowType: &sppb.StructType{
			Fields: []*sppb.StructType_Field{
				{
					Name: "Key",
					Type: &sppb.Type{Code: sppb.TypeCode_STRING},
				},
				{
					Name: "Value",
					Type: &sppb.Type{Code: sppb.TypeCode_STRING},
				},
			},
		},
	}
)

// MockCloudSpanner is a mock implementation of SpannerServer interface.
type MockCloudSpanner struct {
	sppb.SpannerServer

	gsrv *grpc.Server
	lis  net.Listener
	addr string
}

// NewMockCloudSpanner creates a new MockCloudSpanner instance.
func NewMockCloudSpanner() (*MockCloudSpanner, error) {
	gsrv := grpc.NewServer()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	mcs := &MockCloudSpanner{
		gsrv: gsrv,
		lis:  lis,
		addr: lis.Addr().String(),
	}
	sppb.RegisterSpannerServer(gsrv, mcs)
	return mcs, nil
}

// Serve starts the server and blocks.
func (m *MockCloudSpanner) Serve() error {
	return m.gsrv.Serve(m.lis)
}

// Addr returns the listening address of mock server.
func (m *MockCloudSpanner) Addr() string {
	return m.addr
}

// Stop terminates MockCloudSpanner and closes the serving port.
func (m *MockCloudSpanner) Stop() {
	m.gsrv.Stop()
}

// CreateSession is a placeholder for SpannerServer.CreateSession.
func (m *MockCloudSpanner) CreateSession(c context.Context, r *sppb.CreateSessionRequest) (*sppb.Session, error) {
	return &sppb.Session{Name: "some-session"}, nil
}

// DeleteSession is a placeholder for SpannerServer.DeleteSession.
func (m *MockCloudSpanner) DeleteSession(c context.Context, r *sppb.DeleteSessionRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// ExecuteStreamingSql is a mock implementation of SpannerServer.ExecuteStreamingSql.
func (m *MockCloudSpanner) ExecuteStreamingSql(r *sppb.ExecuteSqlRequest, s sppb.Spanner_ExecuteStreamingSqlServer) error {
	rt := EncodeResumeToken(uint64(1))
	meta := KvMeta
	meta.Transaction = &sppb.Transaction{
		ReadTimestamp: &pbt.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   int32(time.Now().Nanosecond()),
		},
	}
	return s.Send(&sppb.PartialResultSet{
		Metadata: &meta,
		Values: []*proto3.Value{
			{Kind: &proto3.Value_StringValue{StringValue: "foo"}},
			{Kind: &proto3.Value_StringValue{StringValue: "bar"}},
		},
		ResumeToken: rt,
	})
}

// StreamingRead is a placeholder for SpannerServer.StreamingRead.
func (m *MockCloudSpanner) StreamingRead(r *sppb.ReadRequest, s sppb.Spanner_StreamingReadServer) error {
	rt := EncodeResumeToken(uint64(1))
	meta := KvMeta
	meta.Transaction = &sppb.Transaction{
		ReadTimestamp: &pbt.Timestamp{
			Seconds: -1,
			Nanos:   -1,
		},
	}
	return s.Send(&sppb.PartialResultSet{
		Metadata: &meta,
		Values: []*proto3.Value{
			{Kind: &proto3.Value_StringValue{StringValue: "foo"}},
			{Kind: &proto3.Value_StringValue{StringValue: "bar"}},
		},
		ResumeToken: rt,
	})
}

// EncodeResumeToken return mock resume token encoding for an uint64 integer.
func EncodeResumeToken(t uint64) []byte {
	rt := make([]byte, 16)
	binary.PutUvarint(rt, t)
	return rt
}
