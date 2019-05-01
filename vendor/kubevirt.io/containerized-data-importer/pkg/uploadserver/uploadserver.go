/*
 * This file is part of the CDI project
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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package uploadserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/pkg/importer"
)

const (
	uploadPath = "/v1alpha1/upload"

	healthzPort = 8080
	healthzPath = "/healthz"
)

// UploadServer is the interface to uploadServerApp
type UploadServer interface {
	Run() error
}

type uploadServerApp struct {
	bindAddress string
	bindPort    int
	destination string
	tlsKey      string
	tlsCert     string
	clientCert  string
	keyFile     string
	certFile    string
	imageSize   string
	mux         *http.ServeMux
	uploading   bool
	done        bool
	doneChan    chan struct{}
	mutex       sync.Mutex
}

// may be overridden in tests
var uploadProcessorFunc = newUploadStreamProcessor

// GetUploadServerURL returns the url the proxy should post to for a particular pvc
func GetUploadServerURL(namespace, pvc string) string {
	return fmt.Sprintf("https://%s.%s.svc%s", controller.GetUploadResourceName(pvc), namespace, uploadPath)
}

// NewUploadServer returns a new instance of uploadServerApp
func NewUploadServer(bindAddress string, bindPort int, destination, tlsKey, tlsCert, clientCert, imageSize string) UploadServer {
	server := &uploadServerApp{
		bindAddress: bindAddress,
		bindPort:    bindPort,
		destination: destination,
		tlsKey:      tlsKey,
		tlsCert:     tlsCert,
		clientCert:  clientCert,
		imageSize:   imageSize,
		mux:         http.NewServeMux(),
		uploading:   false,
		done:        false,
		doneChan:    make(chan struct{}),
	}
	server.mux.HandleFunc(healthzPath, server.healthzHandler)
	server.mux.HandleFunc(uploadPath, server.uploadHandler)
	return server
}

func (app *uploadServerApp) Run() error {
	uploadServer, err := app.createUploadServer()
	if err != nil {
		return errors.Wrap(err, "Error creating upload http server")
	}

	healthzServer, err := app.createHealthzServer()
	if err != nil {
		return errors.Wrap(err, "Error creating healthz http server")
	}

	uploadListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", app.bindAddress, app.bindPort))
	if err != nil {
		return errors.Wrap(err, "Error creating upload listerner")
	}

	healthzListener, err := net.Listen("tcp", fmt.Sprintf(":%d", healthzPort))
	if err != nil {
		return errors.Wrap(err, "Error creating healthz listerner")
	}

	errChan := make(chan error)

	go func() {
		defer uploadListener.Close()

		// maybe bind port was 0 (unit tests) assign port here
		app.bindPort = uploadListener.Addr().(*net.TCPAddr).Port

		if app.keyFile != "" && app.certFile != "" {
			errChan <- uploadServer.ServeTLS(uploadListener, app.certFile, app.keyFile)
			return
		}

		// not sure we want to support this code path
		errChan <- uploadServer.Serve(uploadListener)
	}()

	go func() {
		defer healthzServer.Close()

		errChan <- healthzServer.Serve(healthzListener)
	}()

	select {
	case err = <-errChan:
		klog.Errorf("HTTP server returned error %s", err.Error())
	case <-app.doneChan:
		klog.Info("Shutting down http server after successful upload")
		healthzServer.Shutdown(context.Background())
		uploadServer.Shutdown(context.Background())
	}

	return err
}

func (app *uploadServerApp) createUploadServer() (*http.Server, error) {
	server := &http.Server{
		Handler: app,
	}

	if app.tlsKey != "" && app.tlsCert != "" {
		certDir, err := ioutil.TempDir("", "uploadserver-tls")
		if err != nil {
			return nil, errors.Wrap(err, "Error creating cert dir")
		}

		app.keyFile = filepath.Join(certDir, "tls.key")
		app.certFile = filepath.Join(certDir, "tls.crt")

		err = ioutil.WriteFile(app.keyFile, []byte(app.tlsKey), 0600)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating key file")
		}

		err = ioutil.WriteFile(app.certFile, []byte(app.tlsCert), 0600)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating cert file")
		}
	}

	if app.clientCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(app.clientCert)); !ok {
			klog.Fatalf("Invalid ca cert file %s", app.clientCert)
		}

		server.TLSConfig = &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
	}

	return server, nil
}

func (app *uploadServerApp) createHealthzServer() (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc(healthzPath, app.healthzHandler)
	return &http.Server{Handler: mux}, nil
}

func (app *uploadServerApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *uploadServerApp) healthzHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

func (app *uploadServerApp) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	exit := func() bool {
		app.mutex.Lock()
		defer app.mutex.Unlock()

		if app.uploading {
			w.WriteHeader(http.StatusServiceUnavailable)
			return true
		}

		if app.done {
			w.WriteHeader(http.StatusConflict)
			return true
		}

		app.uploading = true
		return false
	}()

	if exit {
		klog.Warning("Got concurrent upload request")
		return
	}

	err := uploadProcessorFunc(r.Body, app.destination, app.imageSize)

	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err != nil {
		klog.Errorf("Saving stream failed: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		app.uploading = false
		return
	}

	app.uploading = false
	app.done = true

	close(app.doneChan)

	klog.Infof("Wrote data to %s", app.destination)
}

func newUploadStreamProcessor(stream io.ReadCloser, dest, imageSize string) error {
	uds := importer.NewUploadDataSource(stream)
	processor := importer.NewDataProcessor(uds, dest, common.ImporterVolumePath, common.ScratchDataDir, imageSize)
	return processor.ProcessData()
}
