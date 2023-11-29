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
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/tools/cache"
	certificate2 "k8s.io/client-go/util/certificate"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	exportv1 "kubevirt.io/api/export/v1alpha1"
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
	apiVersion         = "v1alpha1"
	exportResourceName = "virtualmachineexports"
	gv                 = apiGroup + "/" + apiVersion
)

type exportProxyApp struct {
	service.ServiceListen
	tlsCertFilePath  string
	tlsKeyFilePath   string
	certManager      certificate2.Manager
	caManager        kvtls.ClientCAManager
	exportInformer   cache.SharedIndexInformer
	kubeVirtInformer cache.SharedIndexInformer
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

	appTLSConfig := kvtls.SetupExportProxyTLS(app.certManager, app.kubeVirtInformer)
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.proxyHandler)
	mux.HandleFunc("/healthz", app.healthzHandler)
	mux.Handle("/metrics", promhttp.Handler())

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

var proxyPathMatcher = regexp.MustCompile(`^/api/` + gv + `/namespaces/([^/]+)/` + exportResourceName + `/([^/]+)/(.*)$`)

func (app *exportProxyApp) proxyHandler(w http.ResponseWriter, r *http.Request) {
	match := proxyPathMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s", match[1], match[2])
	obj, exists, err := app.exportInformer.GetStore().GetByKey(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	export := obj.(*exportv1.VirtualMachineExport)
	if export.Status.Phase != exportv1.Ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	host := fmt.Sprintf("%s.%s.svc:443", export.Status.ServiceName, match[1])
	targetPath := "/" + match[3]

	certPool, err := app.caManager.GetCurrent()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = host
			req.URL.Path = targetPath
			log.Log.Infof("Proxying to %s", req.URL.String())
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	p.ServeHTTP(w, r)
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

	kubeInformerFactory := controller.NewKubeInformerFactory(virtCli.RestClient(), virtCli, aggregatorClient, namespace)
	caInformer := kubeInformerFactory.KubeVirtExportCAConfigMap()
	app.exportInformer = kubeInformerFactory.VirtualMachineExport()
	app.kubeVirtInformer = kubeInformerFactory.KubeVirt()
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
