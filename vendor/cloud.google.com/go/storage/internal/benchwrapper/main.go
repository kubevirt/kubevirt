// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main wraps the client library in a gRPC interface that a benchmarker can communicate through.
package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	"cloud.google.com/go/storage"
	pb "cloud.google.com/go/storage/internal/benchwrapper/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// Ephemeral port to run grpc service.
	port = ":50051"
	// minRead respresents the number of bytes to read at a time.
	minRead = 4
)

func main() {
	ctx := context.Background()
	c, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}
	certificate, err := filepath.Abs("benchwrapper/certificate/server.pem")
	if err != nil {
		log.Fatal(err)
	}
	key, err := filepath.Abs("benchwrapper/certificate/server.key")
	if err != nil {
		log.Fatal(err)
	}
	creds, err := credentials.NewServerTLSFromFile(certificate, key)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterStorageBenchWrapperServer(s, &server{
		c: c,
	})
	log.Printf("Running on %s\n", port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	c *storage.Client
}

func (s *server) Read(ctx context.Context, in *pb.ObjectRead) (*pb.EmptyResponse, error) {
	b := s.c.Bucket(in.GetBucketName())
	o := b.Object(in.GetObjectName())
	r, err := o.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for int(r.Remain()) > 0 {
		ba := make([]byte, minRead)
		_, err := r.Read(ba)
		if err == io.EOF {
			return &pb.EmptyResponse{}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return &pb.EmptyResponse{}, nil
}

func (s *server) Write(ctx context.Context, in *pb.ObjectWrite) (*pb.EmptyResponse, error) {
	b := s.c.Bucket(in.GetBucketName())
	o := b.Object(in.GetObjectName())
	w := o.NewWriter(ctx)
	content, err := ioutil.ReadFile(in.GetDestination())
	if err != nil {
		return nil, err
	}
	w.ContentType = "text/plain"
	if _, err := w.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		if err == io.EOF {
			return &pb.EmptyResponse{}, nil
		}
		return nil, err
	}
	return &pb.EmptyResponse{}, nil
}
