package bootstrap

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net"

	certificates "k8s.io/api/certificates/v1beta1"
	certificatesclient "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	"k8s.io/client-go/util/certificate"
)

func LoadCertConfigForService(certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) *certificate.Config {
	return &certificate.Config{
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				Organization: []string{"kubevirt.io:system"},
				CommonName:   "kubevirt.io:system:services:" + name,
			},
			DNSNames:    dnsSANs,
			IPAddresses: ipSANs,
		},
		CertificateStore: certStore,
		Usages: []certificates.KeyUsage{
			certificates.UsageDigitalSignature,
			certificates.UsageKeyEncipherment,
			certificates.UsageServerAuth,
		},
	}
}

func LoadCertConfigForNode(certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) *certificate.Config {
	return &certificate.Config{
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				Organization: []string{"kubevirt.io:system"},
				CommonName:   "kubevirt.io:system:nodes:" + name,
			},
			DNSNames:    dnsSANs,
			IPAddresses: ipSANs,
		},
		CertificateStore: certStore,
		Usages: []certificates.KeyUsage{
			certificates.UsageDigitalSignature,
			certificates.UsageKeyEncipherment,
			certificates.UsageClientAuth,
			certificates.UsageServerAuth,
		},
	}
}

func NewCertificateManager(config *certificate.Config, bootstrapClient certificatesclient.CertificatesV1beta1Interface) (certificate.Manager, error) {
	manager, err := certificate.NewManager(config)
	if err != nil {
		return nil, err
	}
	err = manager.SetCertificateSigningRequestClient(bootstrapClient.CertificateSigningRequests())
	if err != nil {
		return nil, err
	}
	return manager, nil
}
