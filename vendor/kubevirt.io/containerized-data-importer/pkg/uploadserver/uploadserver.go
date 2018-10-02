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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/pkg/importer"
)

const (
	uploadPath = "/v1alpha1/upload"
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
	mux         *http.ServeMux
	uploading   bool
	done        bool
	doneChan    chan struct{}
	mutex       sync.Mutex
}

// may be overridden in tests
var saveStremFunc = importer.SaveStream

// GetUploadServerURL returns the url the proxy should post to for a particular pvc
func GetUploadServerURL(namespace, pvc string) string {
	return fmt.Sprintf("https://%s.%s.svc%s", controller.GetUploadResourceName(pvc), namespace, uploadPath)
}

// NewUploadServer returns a new instance of uploadServerApp
func NewUploadServer(bindAddress string, bindPort int, destination, tlsKey, tlsCert, clientCert string) UploadServer {
	server := &uploadServerApp{
		bindAddress: bindAddress,
		bindPort:    bindPort,
		destination: destination,
		tlsKey:      tlsKey,
		tlsCert:     tlsCert,
		clientCert:  clientCert,
		mux:         http.NewServeMux(),
		uploading:   false,
		done:        false,
		doneChan:    make(chan struct{}),
	}
	server.mux.HandleFunc(uploadPath, server.uploadHandler)
	return server
}

func (app *uploadServerApp) Run() error {
	var keyFile, certFile string
	server := &http.Server{
		Handler: app,
	}

	if app.tlsKey != "" && app.tlsCert != "" {
		certDir, err := ioutil.TempDir("", "uploadserver-tls")
		if err != nil {
			return errors.Wrap(err, "Error creating cert dir")
		}
		defer os.RemoveAll(certDir)

		keyFile = filepath.Join(certDir, "tls.key")
		certFile = filepath.Join(certDir, "tls.crt")

		err = ioutil.WriteFile(keyFile, []byte(app.tlsKey), 0600)
		if err != nil {
			return errors.Wrap(err, "Error creating key file")
		}

		err = ioutil.WriteFile(certFile, []byte(app.tlsCert), 0600)
		if err != nil {
			return errors.Wrap(err, "Error creating cert file")
		}
	}

	if app.clientCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(app.clientCert)); !ok {
			glog.Fatalf("Invalid ca cert file %s", app.clientCert)
		}

		server.TLSConfig = &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
	}

	errChan := make(chan error)

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", app.bindAddress, app.bindPort))
		if err != nil {
			errChan <- err
			return
		}
		defer listener.Close()

		// maybe bind port was 0 (unit tests) assign port here
		app.bindPort = listener.Addr().(*net.TCPAddr).Port

		if keyFile != "" && certFile != "" {
			errChan <- server.ServeTLS(listener, certFile, keyFile)
			return
		}

		// not sure we want to support this code path
		errChan <- server.Serve(listener)
	}()

	var err error

	select {
	case err = <-errChan:
		glog.Error("HTTP server returned error %s", err.Error())
	case <-app.doneChan:
		glog.Info("Shutting down http server after successful upload")
		server.Shutdown(context.Background())
	}

	return err
}

func (app *uploadServerApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
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
		glog.Warning("Got concurrent upload request")
		return
	}

	sz, err := saveStremFunc(r.Body, app.destination)

	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err != nil {
		glog.Errorf("Saving stream failed: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		app.uploading = false
		return
	}

	app.uploading = false
	app.done = true

	close(app.doneChan)

	glog.Infof("Wrote %d bytes to %s", sz, app.destination)
}
