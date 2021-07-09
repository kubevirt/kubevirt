package webhooks

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"k8s.io/client-go/util/certificate"

	"kubevirt.io/client-go/log"
)

const noSrvCertMessage = "No server certificate, server is not yet ready to receive traffic"

func SetupPromTLS(certManager certificate.Manager) *tls.Config {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, err error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}
			return cert, nil
		},
		GetConfigForClient: func(hi *tls.ClientHelloInfo) (*tls.Config, error) {
			crt := certManager.Current()
			if crt == nil {
				log.Log.Error("failed to get a certificate")
				return nil, fmt.Errorf("failed to get a certificate")
			}
			config := &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: []tls.Certificate{*crt},
				ClientAuth:   tls.VerifyClientCertIfGiven,
			}

			config.BuildNameToCertificate()
			return config, nil
		},
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
func SetupTLSWithCertManager(caManager ClientCAManager, certManager certificate.Manager, clientAuth tls.ClientAuthType) *tls.Config {
	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, err error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}
			return cert, nil
		},
		GetConfigForClient: func(hi *tls.ClientHelloInfo) (*tls.Config, error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}

			clientCAPool, err := caManager.GetCurrent()
			if err != nil {
				log.Log.Reason(err).Error("Failed to get requestheader client CA")
				return nil, err
			}
			config := &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: []tls.Certificate{*cert},
				ClientCAs:    clientCAPool,
				ClientAuth:   clientAuth,
			}

			config.BuildNameToCertificate()
			return config, nil
		},
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

func SetupTLSForVirtHandlerServer(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool) *tls.Config {
	// #nosec cause: InsecureSkipVerify: true
	// resolution: Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
	return &tls.Config{
		//
		InsecureSkipVerify: true,
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, err error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}
			return cert, nil
		},
		GetConfigForClient: func(info *tls.ClientHelloInfo) (config *tls.Config, err error) {
			certPool, err := caManager.GetCurrent()
			if err != nil {
				log.Log.Reason(err).Error("Failed to get kubevirt CA")
				return nil, err
			}
			if certPool == nil {
				return nil, fmt.Errorf("No ca certificate, server is not yet ready to receive traffic")
			}
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}

			config = &tls.Config{
				MinVersion: tls.VersionTLS12,
				ClientCAs:  certPool,
				GetCertificate: func(info *tls.ClientHelloInfo) (i *tls.Certificate, e error) {
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

					rawClient, rawIntermediates := rawCerts[0], rawCerts[1:]
					c, err := x509.ParseCertificate(rawClient)
					if err != nil {
						return fmt.Errorf("failed to parse peer certificate: %v", err)
					}

					intermediatePool := createIntermediatePool(externallyManaged, rawIntermediates)

					_, err = c.Verify(x509.VerifyOptions{
						Roots:         certPool,
						Intermediates: intermediatePool,
						KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					})
					if err != nil {
						return fmt.Errorf("could not verify peer certificate: %v", err)
					}

					if !externallyManaged && c.Subject.CommonName != "kubevirt.io:system:client:virt-handler" {
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

func SetupTLSForVirtHandlerClients(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool) *tls.Config {
	// #nosec cause: InsecureSkipVerify: true
	// resolution: Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
	return &tls.Config{
		// Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, err error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf(noSrvCertMessage)
			}
			return cert, nil
		},
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			cert := certManager.Current()
			if cert == nil {
				return nil, fmt.Errorf("No client certificate, client is not yet ready to talk to the server")
			}
			return cert, nil
		},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			certPool, err := caManager.GetCurrent()
			if err != nil {
				log.Log.Reason(err).Error("Failed to get kubevirt CA")
				return err
			}
			// impossible with RequireAnyClientCert
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided.")
			}

			rawServer, rawIntermediates := rawCerts[0], rawCerts[1:]
			c, err := x509.ParseCertificate(rawServer)
			if err != nil {
				return fmt.Errorf("failed to parse peer certificate: %v", err)
			}

			intermediatePool := createIntermediatePool(externallyManaged, rawIntermediates)

			_, err = c.Verify(x509.VerifyOptions{
				Roots:         certPool,
				Intermediates: intermediatePool,
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			if err != nil {
				return fmt.Errorf("could not verify peer certificate: %v", err)
			}

			if !externallyManaged && c.Subject.CommonName != "kubevirt.io:system:node:virt-handler" {
				return fmt.Errorf("common name is invalid, expected %s, but got %s", "kubevirt.io:system:node:virt-handler", c.Subject.CommonName)
			}

			return nil
		},
	}
}

func createIntermediatePool(externallyManaged bool, rawIntermediates [][]byte) *x509.CertPool {
	var intermediatePool *x509.CertPool = nil
	if externallyManaged {
		intermediatePool = x509.NewCertPool()
		for _, rawIntermediate := range rawIntermediates {
			if c, err := x509.ParseCertificate(rawIntermediate); err != nil {
				log.Log.Warningf("failed to parse peer intermediate certificate: %v", err)
			} else {
				intermediatePool.AddCert(c)
			}
		}
	}
	return intermediatePool
}
