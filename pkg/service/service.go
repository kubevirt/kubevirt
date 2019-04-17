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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package service

import (
	"crypto/tls"
	"crypto/x509"
	goflag "flag"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	flag "github.com/spf13/pflag"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	certificateDir = "/var/lib/kubevirt/certificates"
)

type CertificateConfigCallback func(certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) *certificate.Config

type Service interface {
	Run()
	AddFlags()
	GetName() string
}

type ServiceListen struct {
	Name               string
	BindAddress        string
	Port               int
	CertDir            string
	PromTLSConfig      *tls.Config
	CertificateManager certificate.Manager
	RootCAPool         *x509.CertPool
	PodIpAddress       net.IP
	PodName            string
}

type ServiceLibvirt struct {
	LibvirtUri string
}

func (service *ServiceListen) GetName() string {
	return service.Name
}

func (service *ServiceListen) SetupCertificateManager(virtCli kubecli.KubevirtClient, certificateConfigFunc CertificateConfigCallback) {
	var err error
	service.CertificateManager, service.RootCAPool, err = SetupCertificateManager(service.Name, service.CertDir, service.PodName, service.PodIpAddress, virtCli, certificateConfigFunc)
	if err != nil {
		glog.Fatalf("Failed to setup certificate manager: %v", err)
	}
	go service.CertificateManager.Start()
	service.PromTLSConfig = NewPromTLSConfig(service.Name, service.RootCAPool, service.CertificateManager)
}

func (service *ServiceListen) Address() string {
	return fmt.Sprintf("%s:%s", service.BindAddress, strconv.Itoa(service.Port))
}

func (service *ServiceListen) InitFlags() {
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("kubeconfig"))
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("master"))
}

func (service *ServiceListen) AddCommonFlags() {
	flag.StringVar(&service.BindAddress, "listen", service.BindAddress, "Address where to listen on")
	flag.IntVar(&service.Port, "port", service.Port, "Port to listen on")

	// certificate related flags
	flag.StringVar(&service.CertDir, "cert-dir", certificateDir, "Certificate store directory")
	flag.IPVar(&service.PodIpAddress, "pod-ip-address", net.ParseIP("127.0.0.1"), "The pod ip address")
	flag.StringVar(&service.PodName, "pod-name", "", "The pod name")
}

func (service *ServiceLibvirt) AddLibvirtFlags() {
	flag.StringVar(&service.LibvirtUri, "libvirt-uri", service.LibvirtUri, "Libvirt connection string")

}

func Setup(service Service) {
	log.InitializeLogging(service.GetName())
	service.AddFlags()

	// set new default verbosity, was set to 0 by glog
	flag.Set("v", "2")

	flag.Parse()
}

func SetupCertificateManager(component string, certDir string, podName string, podIP net.IP, virtCli kubecli.KubevirtClient, certificateConfigFunc CertificateConfigCallback) (manager certificate.Manager, rootCA *x509.CertPool, err error) {
	store, err := certificate.NewFileStore("kubevirt-client", certDir, certDir, "", "")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize certificae store: %v", err)
	}
	certExpirationGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: strings.Replace(component, "-", "_", -1),
			Subsystem: "certificate_manager",
			Name:      "client_expiration_seconds",
			Help:      "Gauge of the lifetime of a certificate. The value is the date the certificate will expire in seconds since January 1, 1970 UTC.",
		},
	)
	prometheus.MustRegister(certExpirationGauge)
	config := certificateConfigFunc(store, podName, []string{podName}, []net.IP{podIP})
	config.CertificateExpiration = certExpirationGauge
	manager, err = bootstrap.NewCertificateManager(config, virtCli.CertificatesV1beta1())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup the certificate manager: %v", err)
	}
	certPool, err := certutil.NewPool("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load the root ca: %v", err)
	}
	return manager, certPool, nil
}

func NewPromTLSConfig(component string, certPool *x509.CertPool, manager certificate.Manager) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientCAs:  certPool,
		RootCAs:    certPool,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert := manager.Current()
			if cert == nil {
				return nil, fmt.Errorf("no serving certificate available for %s", component)
			}
			return cert, nil
		},
	}
}
