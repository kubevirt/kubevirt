/*
Copyright 2016 The Kubernetes Authors.

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
	return NewCertificateConfig(certStore, "services:"+name, dnsSANs, ipSANs)
}

func LoadCertConfigForNode(certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) *certificate.Config {
	return NewCertificateConfig(certStore, "nodes:"+name, dnsSANs, ipSANs)
}

func NewCertificateConfig(certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) *certificate.Config {
	return &certificate.Config{
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				Organization: []string{"kubevirt.io:system"},
				CommonName:   "kubevirt.io:system:" + name,
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
