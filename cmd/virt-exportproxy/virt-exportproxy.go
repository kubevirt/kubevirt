package main

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
 * Copyright 2022 Red Hat, Inc.
 *
 */

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/tools/cache"
	certificate2 "k8s.io/client-go/util/certificate"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/service"
)

const (
	defaultTlsCertFilePath = "/etc/virt-exportproxy/certificates/tls.crt"
	defaultTlsKeyFilePath  = "/etc/virt-exportproxy/certificates/tls.key"

	apiGroup           = "export.kubevirt.io"
	apiVersions        = "v1beta1|v1"
	exportResourceName = "virtualmachineexports"

	backendIdleConnTimeout       = 30 * time.Second
	backendDialTimeout           = 10 * time.Second
	backendDialKeepAlive         = 30 * time.Second
	backendResponseHeaderTimeout = 30 * time.Second
	serverIdleTimeout            = 60 * time.Second
	serverReadHeaderTimeout      = 10 * time.Second
)

type exportProxyApp struct {
	service.ServiceListen
	tlsCertFilePath string
	tlsKeyFilePath  string
	certManager     certificate2.Manager
	caManager       kvtls.ClientCAManager
	exportStore     cache.Store
	kubeVirtStore   cache.Store
	// reverseProxy is a shared template; proxyHandler takes a shallow copy per
	// request and sets a per-request Rewrite closure on the copy.
	reverseProxy *httputil.ReverseProxy
}

func NewExportProxyApp() service.Service {
	return &exportProxyApp{}
}

func (app *exportProxyApp) AddFlags() {
	app.InitFlags()
	app.AddCommonFlags()

	flag.StringVar(&app.tlsCertFilePath, "tls-cert-file", defaultTlsCertFilePath,
		"File containing the default x509 Certificate for HTTPS")
	flag.StringVar(&app.tlsKeyFilePath, "tls-key-file", defaultTlsKeyFilePath,
		"File containing the default x509 private key matching --tls-cert-file")
}

func (app *exportProxyApp) Run() {
	stopChan := make(chan struct{}, 1)
	defer close(stopChan)
	if err := app.prepareInformers(stopChan); err != nil {
		panic(err)
	}

	app.prepareCertManager()
	go app.certManager.Start()

	app.initReverseProxy()

	appTLSConfig := kvtls.SetupExportProxyTLS(app.certManager, app.kubeVirtStore)
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.proxyHandler)
	mux.HandleFunc("/healthz", app.healthzHandler)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              app.Address(),
		Handler:           mux,
		TLSConfig:         appTLSConfig,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		IdleTimeout:       serverIdleTimeout,
		// Disable HTTP/2
		// See CVE-2023-44487
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		panic(err)
	}
}

func (app *exportProxyApp) healthzHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

var proxyPathMatcher = regexp.MustCompile(`^/api/` + apiGroup + "/" + "(" + apiVersions + ")" + `/namespaces/([^/]+)/` + exportResourceName + `/([^/]+)/(.*)$`)

func (app *exportProxyApp) proxyHandler(w http.ResponseWriter, r *http.Request) {
	match := proxyPathMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s", match[2], match[3])
	obj, exists, err := app.exportStore.GetByKey(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	export := obj.(*exportv1.VirtualMachineExport)
	if export.Status == nil || export.Status.Phase != exportv1.Ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	backendHost := fmt.Sprintf("%s.%s.svc:443", export.Status.ServiceName, match[2])
	backendPath := "/" + match[4]
	log.Log.V(4).Infof("Proxying to https://%s%s", backendHost, backendPath)
	proxy := *app.reverseProxy
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		// Route via Out (not SetURL) so the inbound path is not joined onto the target.
		pr.Out.URL.Scheme = "https"
		pr.Out.URL.Host = backendHost
		pr.Out.URL.Path = backendPath
		pr.Out.URL.RawPath = ""
		pr.Out.Host = ""
	}
	proxy.ServeHTTP(w, r)
}

func (app *exportProxyApp) initReverseProxy() {
	transport := &http.Transport{
		DialTLSContext:        app.dialBackendTLS,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       backendIdleConnTimeout,
		ResponseHeaderTimeout: backendResponseHeaderTimeout,
	}
	app.reverseProxy = &httputil.ReverseProxy{
		Transport:     transport,
		FlushInterval: -1, // flush immediately; avoids proxy-side buffering of large export streams
	}
}

func (app *exportProxyApp) dialBackendTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   backendDialTimeout,
		KeepAlive: backendDialKeepAlive,
	}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	serverName, _, err := net.SplitHostPort(addr)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("could not parse backend address %q: %w", addr, err)
	}
	cfg := &tls.Config{
		// Neither the client nor the server should validate anything itself; VerifyConnection is still executed.
		InsecureSkipVerify: true, // #nosec G402 -- VerifyConnection performs certificate verification
		VerifyConnection:   app.verifyBackendConnection,
		ServerName:         serverName,
	}

	tlsConn := tls.Client(conn, cfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func (app *exportProxyApp) verifyBackendConnection(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("backend presented no certificate")
	}
	if cs.ServerName == "" {
		return fmt.Errorf("backend TLS ServerName is required")
	}

	certPool, err := app.caManager.GetCurrent()
	if err != nil {
		return err
	}

	peer := cs.PeerCertificates[0]
	intermediates := x509.NewCertPool()
	for _, intermediate := range cs.PeerCertificates[1:] {
		intermediates.AddCert(intermediate)
	}

	opts := x509.VerifyOptions{
		Roots:         certPool,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSName:       cs.ServerName,
	}

	_, err = peer.Verify(opts)
	if err != nil {
		return fmt.Errorf("could not verify backend certificate: %w", err)
	}
	return nil
}

func (app *exportProxyApp) prepareInformers(stopChan <-chan struct{}) error {
	namespace, err := clientutil.GetNamespace()
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubevirt client config: %w", err)
	}
	virtCli, err := kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubevirt client: %w", err)
	}
	aggregatorClient := aggregatorclient.NewForConfigOrDie(clientConfig)

	kubeInformerFactory := controller.NewKubeInformerFactory(virtCli.RestClient(), virtCli, virtCli, aggregatorClient, namespace)
	caInformer := kubeInformerFactory.KubeVirtExportCAConfigMap()
	app.exportStore = kubeInformerFactory.VirtualMachineExport().GetStore()
	app.kubeVirtStore = kubeInformerFactory.KubeVirt().GetStore()
	kubeInformerFactory.Start(stopChan)
	kubeInformerFactory.WaitForCacheSync(stopChan)

	app.caManager = kvtls.NewCAManager(caInformer.GetStore(), namespace, "kubevirt-export-ca")
	return nil
}

func (app *exportProxyApp) prepareCertManager() {
	app.certManager = bootstrap.NewFileCertificateManager(app.tlsCertFilePath, app.tlsKeyFilePath)
}

func main() {
	log.InitializeLogging("virt-exportproxy")
	log.Log.Info("Starting export proxy")

	app := NewExportProxyApp()
	service.Setup(app)
	app.Run()
}
