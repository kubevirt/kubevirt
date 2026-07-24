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
	"strconv"
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
	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
	exportproxymetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-exportproxy"
	"kubevirt.io/kubevirt/pkg/service"
)

const (
	defaultTlsCertFilePath = "/etc/virt-exportproxy/certificates/tls.crt"
	defaultTlsKeyFilePath  = "/etc/virt-exportproxy/certificates/tls.key"

	apiGroup           = "export.kubevirt.io"
	apiVersions        = "v1beta1|v1"
	exportResourceName = "virtualmachineexports"

	backendIdleConnTimeout       = 90 * time.Second
	backendDialTimeout           = 10 * time.Second
	backendDialKeepAlive         = 30 * time.Second
	backendResponseHeaderTimeout = 30 * time.Second
	serverIdleTimeout            = 60 * time.Second
	serverReadHeaderTimeout      = 10 * time.Second

	directorErrorHost = "virt-exportproxy-director-error.invalid"

	proxyRateLimitedBody = "rate limited"
)

type proxyTargetContextKey struct{}

var proxyTargetContextKeyVar = proxyTargetContextKey{}

type proxyTarget struct {
	host string
	path string
}

type exportProxyApp struct {
	service.ServiceListen
	tlsCertFilePath string
	tlsKeyFilePath  string
	certManager     certificate2.Manager
	caManager       kvtls.ClientCAManager
	exportStore     cache.Store
	kubeVirtStore   cache.Store
	proxyTransport  *http.Transport
	reverseProxy    *httputil.ReverseProxy
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
	app.prepareInformers(stopChan)

	app.prepareCertManager()
	go app.certManager.Start()

	app.initReverseProxy()

	if err := exportproxymetrics.SetupMetrics(); err != nil {
		panic(err)
	}

	appTLSConfig := kvtls.SetupExportProxyTLS(app.certManager, app.kubeVirtStore)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", app.healthzHandler)
	mux.HandleFunc("/readyz", app.readyzHandler)
	mux.HandleFunc("/api/", app.proxyHandler)

	server := &http.Server{
		Addr:      app.Address(),
		Handler:   mux,
		TLSConfig: appTLSConfig,
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

func (app *exportProxyApp) readyzHandler(w http.ResponseWriter, r *http.Request) {
	exportproxymetrics.WriteReadyzResponse(w)
}

var proxyPathMatcher = regexp.MustCompile(`^/api/` + apiGroup + "/" + "(" + apiVersions + ")" + `/namespaces/([^/]+)/` + exportResourceName + `/([^/]+)/(.*)$`)

func (app *exportProxyApp) proxyHandler(w http.ResponseWriter, r *http.Request) {
	match := proxyPathMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	namespace := match[2]
	exportName := match[3]
	targetPath := "/" + match[4]

	activeTransfer, ok := exportproxymetrics.TryRecordTransferStarted()
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(admission.RetryAfterSeconds))
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, proxyRateLimitedBody)
		return
	}
	defer activeTransfer.Finish()

	serviceName, ready, err := app.resolveServiceName(namespace, exportName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ready {
		if serviceName == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	host := fmt.Sprintf("%s.%s.svc:443", serviceName, namespace)

	ctx := context.WithValue(r.Context(), proxyTargetContextKeyVar, proxyTarget{
		host: host,
		path: targetPath,
	})
	app.reverseProxy.ServeHTTP(w, r.WithContext(ctx))
}

func (app *exportProxyApp) initReverseProxy() {
	app.proxyTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   backendDialTimeout,
			KeepAlive: backendDialKeepAlive,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// #nosec cause: InsecureSkipVerify: true
			// resolution: Neither the client nor the server should validate anything itself, `VerifyConnection` is still executed
			InsecureSkipVerify: true,
			VerifyConnection:   app.verifyBackendConnection,
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       backendIdleConnTimeout,
		ResponseHeaderTimeout: backendResponseHeaderTimeout,
	}
	app.reverseProxy = &httputil.ReverseProxy{
		Transport:      app.proxyTransport,
		Director:       app.proxyDirector,
		ErrorHandler:   app.proxyErrorHandler,
		ModifyResponse: app.modifyProxyResponse,
	}
}

func (app *exportProxyApp) modifyProxyResponse(resp *http.Response) error {
	resp.Body = exportproxymetrics.NewCountingReadCloser(resp.Body)
	return nil
}

func (app *exportProxyApp) verifyBackendConnection(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("backend presented no certificate")
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
	}
	if cs.ServerName == "" {
		return fmt.Errorf("backend TLS ServerName is required")
	}
	opts.DNSName = cs.ServerName

	_, err = peer.Verify(opts)
	if err != nil {
		return fmt.Errorf("could not verify backend certificate: %w", err)
	}
	return nil
}

func (app *exportProxyApp) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if r != nil && r.URL != nil && r.URL.Host == directorErrorHost {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	log.Log.Reason(err).Error("failed to proxy export backend request")
	http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
}

func (app *exportProxyApp) proxyDirector(req *http.Request) {
	target, ok := req.Context().Value(proxyTargetContextKeyVar).(proxyTarget)
	if !ok {
		log.Log.Error("proxyDirector: missing proxyTarget in request context")
		req.URL.Scheme = "https"
		req.URL.Host = directorErrorHost
		req.URL.Path = "/"
		return
	}

	req.URL.Scheme = "https"
	req.URL.Host = target.host
	req.URL.Path = target.path
	log.Log.V(4).Infof("Proxying to %s", req.URL.String())
	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func (app *exportProxyApp) resolveServiceName(namespace, exportName string) (serviceName string, ready bool, err error) {
	key := fmt.Sprintf("%s/%s", namespace, exportName)
	obj, exists, err := app.exportStore.GetByKey(key)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}

	export := obj.(*exportv1.VirtualMachineExport)
	if export.Status.Phase != exportv1.Ready {
		return export.Status.ServiceName, false, nil
	}
	return export.Status.ServiceName, true, nil
}

func (app *exportProxyApp) prepareInformers(stopChan <-chan struct{}) {
	namespace, err := clientutil.GetNamespace()
	if err != nil {
		panic(err)
	}

	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		panic(err)
	}
	virtCli, err := kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		panic(err)
	}
	aggregatorClient := aggregatorclient.NewForConfigOrDie(clientConfig)

	kubeInformerFactory := controller.NewKubeInformerFactory(virtCli.RestClient(), virtCli, virtCli, aggregatorClient, namespace)
	caInformer := kubeInformerFactory.KubeVirtExportCAConfigMap()
	app.exportStore = kubeInformerFactory.VirtualMachineExport().GetStore()
	app.kubeVirtStore = kubeInformerFactory.KubeVirt().GetStore()
	kubeInformerFactory.Start(stopChan)
	kubeInformerFactory.WaitForCacheSync(stopChan)

	app.caManager = kvtls.NewCAManager(caInformer.GetStore(), namespace, "kubevirt-export-ca")
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
