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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	certificate2 "k8s.io/client-go/util/certificate"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
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
	tlsCertFilePath string
	tlsKeyFilePath  string
	certManager     certificate2.Manager
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
	app.prepareCertManager()

	go app.certManager.Start()

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.proxyHandler)
	mux.HandleFunc("/healthz", app.healthzHandler)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    app.Address(),
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, err error) {
				cert := app.certManager.Current()
				if cert == nil {
					return nil, fmt.Errorf("error getting cert")
				}
				return cert, nil
			},
		},
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

	// TODO lookup export resource and get service
	host := fmt.Sprintf("virt-export-%s.%s.svc:443", match[2], match[1])
	targetPath := "/" + match[3]

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
		// TODO handle server ca
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	p.ServeHTTP(w, r)
}

func (app *exportProxyApp) prepareCertManager() {
	app.certManager = bootstrap.NewFileCertificateManager(app.tlsCertFilePath, app.tlsKeyFilePath)
	//app.handlerCertManager = bootstrap.NewFileCertificateManager(app.handlerCertFilePath, app.handlerKeyFilePath)
}

func main() {
	log.InitializeLogging("virt-exportproxy")
	log.Log.Info("Starting export proxy")

	app := NewExportProxyApp()
	service.Setup(app)
	app.Run()
}
