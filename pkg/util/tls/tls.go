package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"k8s.io/client-go/util/certificate"

	"kubevirt.io/client-go/log"
)

const noSrvCertMessage = "No server certificate, server is not yet ready to receive traffic"
const serverNotReadyMsg = "Server is not yet ready to receive traffic"

var (
	cipherSuites         = tls.CipherSuites()
	insecureCipherSuites = tls.InsecureCipherSuites()
)

func SetupPromTLS(certManager certificate.Manager, clusterConfig *virtconfig.ClusterConfig) *tls.Config {
	tlsConfig := &tls.Config{
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
				log.Log.Error(noSrvCertMessage)
				return nil, fmt.Errorf(noSrvCertMessage)
			}

			kv := clusterConfig.GetConfigFromKubeVirtCR()
			tlsConfig := getTLSConfiguration(kv)
			ciphers := CipherSuiteIds(tlsConfig.Ciphers)
			minTLSVersion := TLSVersion(tlsConfig.MinTLSVersion)
			config := &tls.Config{
				CipherSuites: ciphers,
				MinVersion:   minTLSVersion,
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

func SetupExportProxyTLS(certManager certificate.Manager, kubeVirtStore cache.Store) *tls.Config {
	tlsConfig := &tls.Config{
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
				log.Log.Error(noSrvCertMessage)
				return nil, fmt.Errorf(noSrvCertMessage)
			}

			kv := getKubevirt(kubeVirtStore)
			tlsConfig := getTLSConfiguration(kv)
			ciphers := CipherSuiteIds(tlsConfig.Ciphers)
			minTLSVersion := TLSVersion(tlsConfig.MinTLSVersion)
			config := &tls.Config{
				CipherSuites: ciphers,
				MinVersion:   minTLSVersion,
				Certificates: []tls.Certificate{*crt},
			}

			config.BuildNameToCertificate()
			return config, nil
		},
	}
	return tlsConfig
}

func SetupTLSWithCertManager(caManager KubernetesCAManager, certManager certificate.Manager, clientAuth tls.ClientAuthType, clusterConfig *virtconfig.ClusterConfig) *tls.Config {
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

			kv := clusterConfig.GetConfigFromKubeVirtCR()
			tlsConfig := getTLSConfiguration(kv)
			ciphers := CipherSuiteIds(tlsConfig.Ciphers)
			minTLSVersion := TLSVersion(tlsConfig.MinTLSVersion)
			config := &tls.Config{
				CipherSuites: ciphers,
				MinVersion:   minTLSVersion,
				Certificates: []tls.Certificate{*cert},
				ClientCAs:    clientCAPool,
				ClientAuth:   clientAuth,
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
						return nil
					}

					certificate, err := x509.ParseCertificate(rawCerts[0])
					if err != nil {
						return fmt.Errorf("failed to parse peer certificate: %v", err)
					}

					CNs, err := caManager.GetCNs()
					if err != nil {
						log.Log.Reason(err).Error(serverNotReadyMsg)
						return fmt.Errorf(serverNotReadyMsg)
					}

					if len(CNs) == 0 {
						return nil
					}
					for _, CN := range CNs {
						if certificate.Subject.CommonName == CN {
							return nil
						}
					}

					return fmt.Errorf("Common name is invalid")
				},
			}

			config.BuildNameToCertificate()
			return config, nil
		},
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

func SetupTLSForVirtSynchronizationControllerServer(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool, clusterConfig *virtconfig.ClusterConfig) *tls.Config {
	return SetupTLSForServer(caManager, certManager, externallyManaged, clusterConfig, "virt-synchronization-controller")
}

func SetupTLSForVirtHandlerServer(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool, clusterConfig *virtconfig.ClusterConfig) *tls.Config {
	return SetupTLSForServer(caManager, certManager, externallyManaged, clusterConfig, "virt-handler")
}

func SetupTLSForServer(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool, clusterConfig *virtconfig.ClusterConfig, commonNameType string) *tls.Config {
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

			kv := clusterConfig.GetConfigFromKubeVirtCR()
			tlsConfig := getTLSConfiguration(kv)
			ciphers := CipherSuiteIds(tlsConfig.Ciphers)
			minTLSVersion := TLSVersion(tlsConfig.MinTLSVersion)
			config = &tls.Config{
				CipherSuites: ciphers,
				MinVersion:   minTLSVersion,
				ClientCAs:    certPool,
				GetCertificate: func(info *tls.ClientHelloInfo) (i *tls.Certificate, e error) {
					return cert, nil
				},
				// Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
				InsecureSkipVerify: true,
				// XXX: We need to verify the cert ourselves because we don't have DNS or IP on the certs at the moment
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					return verifyPeerCert(rawCerts, externallyManaged, certPool, x509.ExtKeyUsageClientAuth, "client", commonNameType)
				},
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			return config, nil
		},
	}
}

func SetupTLSForVirtSynchronizationControllerClients(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool) *tls.Config {
	return SetupTLSForClients(caManager, certManager, externallyManaged, "virt-synchronization-controller")
}

func SetupTLSForVirtHandlerClients(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool) *tls.Config {
	return SetupTLSForClients(caManager, certManager, externallyManaged, "virt-handler")
}

func SetupTLSForClients(caManager ClientCAManager, certManager certificate.Manager, externallyManaged bool, commonNameType string) *tls.Config {
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
			return verifyPeerCert(rawCerts, externallyManaged, certPool, x509.ExtKeyUsageServerAuth, "node", commonNameType)
		},
	}
}

func getTLSConfiguration(kubevirt *v1.KubeVirt) *v1.TLSConfiguration {
	tlsConfiguration := &v1.TLSConfiguration{
		MinTLSVersion: v1.VersionTLS12,
		Ciphers:       nil,
	}

	if kubevirt != nil && kubevirt.Spec.Configuration.TLSConfiguration != nil {
		tlsConfiguration = kubevirt.Spec.Configuration.TLSConfiguration
	}
	return tlsConfiguration
}

func CipherSuiteIds(names []string) []uint16 {
	var idByName = CipherSuiteNameMap()
	var ids []uint16
	for _, name := range names {
		if id, ok := idByName[name]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func CipherSuiteNameMap() map[string]uint16 {
	var idByName = map[string]uint16{}
	for _, cipherSuite := range cipherSuites {
		idByName[cipherSuite.Name] = cipherSuite.ID
	}
	for _, cipherSuite := range insecureCipherSuites {
		idByName[cipherSuite.Name] = cipherSuite.ID
	}
	return idByName
}

// TLSVersion converts from human-readable TLS version (for example "1.1")
// to the values accepted by tls.Config (for example 0x301).
func TLSVersion(version v1.TLSProtocolVersion) uint16 {
	switch version {
	case v1.VersionTLS10:
		return tls.VersionTLS10
	case v1.VersionTLS11:
		return tls.VersionTLS11
	case v1.VersionTLS12:
		return tls.VersionTLS12
	case v1.VersionTLS13:
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12
	}
}

// TLSVersionName converts from tls.Config id version to human-readable TLS version.
func TLSVersionName(versionId uint16) string {
	switch versionId {
	case tls.VersionTLS10:
		return string(v1.VersionTLS10)
	case tls.VersionTLS11:
		return string(v1.VersionTLS11)
	case tls.VersionTLS12:
		return string(v1.VersionTLS12)
	case tls.VersionTLS13:
		return string(v1.VersionTLS13)
	default:
		return fmt.Sprintf("0x%04X", versionId)
	}
}

func verifyPeerCert(rawCerts [][]byte, externallyManaged bool, certPool *x509.CertPool, usage x509.ExtKeyUsage, commonName, commonNameType string) error {
	// impossible with RequireAnyClientCert
	if len(rawCerts) == 0 {
		return fmt.Errorf("no client certificate provided.")
	}

	rawPeer, rawIntermediates := rawCerts[0], rawCerts[1:]
	c, err := x509.ParseCertificate(rawPeer)
	if err != nil {
		return fmt.Errorf("failed to parse peer certificate: %v", err)
	}

	intermediatePool := createIntermediatePool(externallyManaged, rawIntermediates)

	_, err = c.Verify(x509.VerifyOptions{
		Roots:         certPool,
		Intermediates: intermediatePool,
		KeyUsages:     []x509.ExtKeyUsage{usage},
	})
	if err != nil {
		return fmt.Errorf("could not verify peer certificate: %v", err)
	}

	fullCommonName := fmt.Sprintf("kubevirt.io:system:%s:%s", commonName, commonNameType)
	if !externallyManaged && c.Subject.CommonName != fullCommonName {
		return fmt.Errorf("common name is invalid, expected %s, but got %s", fullCommonName, c.Subject.CommonName)
	}

	return nil
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

func getKubevirt(kubeVirtStore cache.Store) *v1.KubeVirt {
	objects := kubeVirtStore.List()
	for _, obj := range objects {
		if kv, ok := obj.(*v1.KubeVirt); ok && kv.DeletionTimestamp == nil {
			if kv.Status.Phase != "" {
				return obj.(*v1.KubeVirt)
			}
		}
	}
	return nil
}
