package webhooks

import (
	"crypto/tls"

	"kubevirt.io/client-go/log"
)

func SetupTLS(caManager ClientCAManager, serverCert tls.Certificate, clientAuth tls.ClientAuthType) *tls.Config {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		GetConfigForClient: func(hi *tls.ClientHelloInfo) (*tls.Config, error) {

			pool, err := caManager.GetCurrent()
			if err != nil {
				log.Log.Reason(err).Error("Failed to get requestheader client CA")
				return nil, err
			}
			config := &tls.Config{
				Certificates: []tls.Certificate{serverCert},
				ClientCAs:    pool,
				ClientAuth:   clientAuth,
			}

			config.BuildNameToCertificate()
			return config, nil
		},
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
