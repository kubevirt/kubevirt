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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"k8s.io/client-go/util/keyutil"

	"github.com/mdlayher/vsock"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	v1 "kubevirt.io/kubevirt/pkg/vsock/system/v1"
)

func main() {
	log.InitializeLogging("example-guest-agent")

	var useTLS bool
	var port uint32
	pflag.BoolVar(&useTLS, "use-tls", false, "weather to use pick up the kubevirt CA and use it for TLS authentication of incoming connections")
	pflag.Uint32Var(&port, "port", 1234, "vsock port to listen on")
	pflag.Parse()

	vsockConn, err := vsock.Listen(port, &vsock.Config{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to start vsock server")
		os.Exit(1)
	}
	defer vsockConn.Close()

	var serverConn net.Listener
	if useTLS {
		log.DefaultLogger().Infof("Using TLS ...")
		serverCert, err := setupCert()
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("could not generate the server TLS certificate")
			os.Exit(1)
		}
		tlsConfig := SetupTLSForVirtHandlerServer(serverCert)
		serverConn = tls.NewListener(vsockConn, tlsConfig)
	} else {
		log.DefaultLogger().Infof("Not using TLS ...")
		serverConn = vsockConn
	}
	defer serverConn.Close()

	log.DefaultLogger().Infof("starting listening")
	for {
		func() {
			c, err := serverConn.Accept()
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("lala")
				return
			}
			defer c.Close()
			log.DefaultLogger().Infof("connection accepted")
			buf := make([]byte, 12)
			n, err := c.Read(buf)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed to read the message")
				return
			}

			if _, err = c.Write(buf[0:n]); err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed to mirror the received message")
				return
			}
		}()
	}
}

func refreshBundle(service v1.SystemClient) (*x509.CertPool, error) {
	bundle, err := service.CABundle(context.Background(), &v1.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to call service: %v", err)
	}

	crt, err := cert.ParseCertsPEM(bundle.Raw)
	if err != nil {
		return nil, fmt.Errorf("invalid ca bundle received: %v", err)
	}
	pool := x509.NewCertPool()
	for _, crt := range crt {
		pool.AddCert(crt)
	}
	log.DefaultLogger().Infof("loaded a bundle containing %v certificates", len(crt))
	return pool, nil
}

func setupCert() (*tls.Certificate, error) {
	ca, err := triple.NewCA("test", 1000*time.Hour)
	if err != nil {
		return nil, err
	}
	pair, err := triple.NewServerKeyPair(ca, "test", "test", "test", "test", nil, nil, 1*time.Hour)
	if err != nil {
		return nil, err
	}
	pem, err := keyutil.MarshalPrivateKeyToPEM(pair.Key)
	if err != nil {
		return nil, err
	}
	crt, err := tls.X509KeyPair(cert.EncodeCertPEM(pair.Cert), pem)
	if err != nil {
		return nil, err
	}
	return &crt, nil
}

func SetupTLSForVirtHandlerServer(cert *tls.Certificate) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert, nil
		},
		GetConfigForClient: func(info *tls.ClientHelloInfo) (config *tls.Config, err error) {
			dialOpt := grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
				return vsock.Dial(vsock.Host, 1, &vsock.Config{})
			})
			conn, err := grpc.Dial("something", dialOpt, grpc.WithInsecure())
			if err != nil {
				return nil, fmt.Errorf("failed to connect to the system service: %v", err)
			}
			defer conn.Close()

			certPool, err := refreshBundle(v1.NewSystemClient(conn))
			if err != nil {
				return nil, err
			}

			config = &tls.Config{
				MinVersion: tls.VersionTLS13,
				ClientCAs:  certPool,
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					return cert, nil
				},
				// Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
				InsecureSkipVerify: true,
				// XXX: We need to verify the cert ourselves because we don't have DNS or IP on the certs at the moment
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					// impossible with RequireAnyClientCert
					if len(rawCerts) == 0 {
						return fmt.Errorf("no client certificate provided.")
					}

					c, err := x509.ParseCertificate(rawCerts[0])
					if err != nil {
						return fmt.Errorf("failed to parse peer certificate: %v", err)
					}

					log.DefaultLogger().Infof("rereshed CA bundle")
					_, err = c.Verify(x509.VerifyOptions{
						Roots:     certPool,
						KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					})
					if err != nil {
						return fmt.Errorf("could not verify peer certificate: %v", err)
					}

					if c.Subject.CommonName != "kubevirt.io:system:client:virt-handler" {
						return fmt.Errorf("common name is invalid, expected %s, but got %s", "kubevirt.io:system:client:virt-handler", c.Subject.CommonName)
					}

					return nil
				},
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			return config, nil
		},
	}
}
