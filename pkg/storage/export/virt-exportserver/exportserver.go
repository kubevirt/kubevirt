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

package virtexportserver

import (
	"bytes"
	"compress/gzip"
	"context"
	goflag "flag"
	"io"
	"io/ioutil"
	golog "log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/service"
)

const (
	authHeader = "x-kubevirt-export-token"
)

type TokenGetterFunc func() (string, error)

type VolumeInfo struct {
	Path       string
	ArchiveURI string
	DirURI     string
	RawURI     string
	RawGzURI   string
}
type ExportServerConfig struct {
	Deadline time.Time

	ListenAddr string

	CertFile, KeyFile string

	TokenFile string

	Volumes []VolumeInfo

	// unit testing helpers
	ArchiveHandler func(string) http.Handler
	DirHandler     func(string, string) http.Handler
	FileHandler    func(string) http.Handler
	GzipHandler    func(string) http.Handler

	TokenGetter TokenGetterFunc
}

type execReader struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

type exportServer struct {
	ExportServerConfig
	handler http.Handler
}

func (er *execReader) Read(p []byte) (int, error) {
	n, err := er.stdout.Read(p)
	if err == io.EOF {
		if err2 := er.cmd.Wait(); err2 != nil {
			errBytes, _ := ioutil.ReadAll(er.stderr)
			log.Log.Reason(err2).Errorf("Subprocess did not execute successfully, result is: %q\n%s", er.cmd.ProcessState.ExitCode(), string(errBytes))
			return n, err2
		}
	}
	return n, err
}

func (er *execReader) Close() error {
	return er.stdout.Close()
}

func (s *exportServer) initHandler() {
	mux := http.NewServeMux()
	for _, vi := range s.Volumes {
		if hasPermissions := checkVolumePermissions(vi.Path); !hasPermissions {
			golog.Fatalf("unable to manipulate %s's contents, exiting", vi.Path)
		}
		for path, handler := range s.getHandlerMap(vi) {
			log.Log.Infof("Handling path %s\n", path)
			mux.Handle(path, tokenChecker(s.TokenGetter, handler))
		}
	}

	s.handler = mux
}

func (s *exportServer) getHandlerMap(vi VolumeInfo) map[string]http.Handler {
	fi, err := os.Stat(vi.Path)
	if err != nil {
		log.Log.Reason(err).Errorf("error statting %s", vi.Path)
		return nil
	}

	var result = make(map[string]http.Handler)

	if vi.ArchiveURI != "" {
		result[vi.ArchiveURI] = s.ArchiveHandler(vi.Path)
	}

	if vi.DirURI != "" {
		result[vi.DirURI] = s.DirHandler(vi.DirURI, vi.Path)
	}

	p := vi.Path
	if fi.IsDir() {
		p = path.Join(p, "disk.img")
	}

	if vi.RawURI != "" {
		result[vi.RawURI] = s.FileHandler(p)
	}

	if vi.RawGzURI != "" {
		result[vi.RawGzURI] = s.GzipHandler(p)
	}

	return result
}

func (s *exportServer) Run() {
	s.initHandler()

	srv := &http.Server{
		Addr:    s.ListenAddr,
		Handler: s.handler,
	}

	ch := make(chan error)

	go func() {
		err := srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
		ch <- err
	}()

	if !s.Deadline.IsZero() {
		log.Log.Infof("Deadline set to %s", s.Deadline)
		select {
		case err := <-ch:
			panic(err)
		case <-time.After(time.Until(s.Deadline)):
			log.Log.Info("Deadline exceeded, shutting down")
			srv.Shutdown(context.TODO())
		}
	} else {
		err := <-ch
		panic(err)
	}
}

func (s *exportServer) AddFlags() {
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
}

func NewExportServer(config ExportServerConfig) service.Service {
	es := &exportServer{ExportServerConfig: config}

	if es.ArchiveHandler == nil {
		es.ArchiveHandler = archiveHandler
	}

	if es.DirHandler == nil {
		es.DirHandler = dirHandler
	}

	if es.FileHandler == nil {
		es.FileHandler = fileHandler
	}

	if es.GzipHandler == nil {
		es.GzipHandler = gzipHandler
	}

	if es.TokenGetter == nil {
		es.TokenGetter = func() (string, error) {
			return getToken(es.TokenFile)
		}
	}

	return es
}

func newTarReader(mountPoint string) (io.ReadCloser, error) {
	cmd := exec.Command("/usr/bin/tar", "Scv", ".")
	cmd.Dir = mountPoint

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	return &execReader{cmd: cmd, stdout: stdout, stderr: ioutil.NopCloser(&stderr)}, nil
}

func pipeToGzip(reader io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	zw := gzip.NewWriter(pw)

	go func() {
		n, err := io.Copy(zw, reader)
		if err != nil {
			log.Log.Reason(err).Error("error piping to gzip")
		}
		if err = zw.Close(); err != nil {
			log.Log.Reason(err).Error("error closing gzip writer")
		}
		if err = pw.Close(); err != nil {
			log.Log.Reason(err).Error("error closing pipe writer")
		}
		log.Log.Infof("Wrote %d bytes\n", n)
	}()

	return pr
}

func getTokenQueryParam(r *http.Request) (token string) {
	q := r.URL.Query()
	if keys, ok := q[authHeader]; ok {
		token = keys[0]
		q.Del(authHeader)
		r.URL.RawQuery = q.Encode()
	}
	return
}

func getTokenHeader(r *http.Request) (token string) {
	if tok := r.Header.Get(authHeader); tok != "" {
		r.Header.Del(authHeader)
		token = tok
	}
	return
}

func tokenChecker(tokenGetter TokenGetterFunc, nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := tokenGetter()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, tok := range []string{getTokenQueryParam(r), getTokenHeader(r)} {
			if tok == token {
				nextHandler.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func archiveHandler(mountPoint string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if hasPermissions := checkDirectoryPermissions(mountPoint); !hasPermissions {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		tarReader, err := newTarReader(mountPoint)
		if err != nil {
			log.Log.Reason(err).Error("error creating tar reader")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer tarReader.Close()
		gzipReader := pipeToGzip(tarReader)
		defer gzipReader.Close()
		n, err := io.Copy(w, gzipReader)
		if err != nil {
			log.Log.Reason(err).Error("error writing response body")
		}
		log.Log.Infof("Wrote %d bytes\n", n)
	})
}

func checkDirectoryPermissions(filePath string) bool {
	dir, err := os.Open(filePath)
	if err != nil {
		log.Log.Reason(err).Errorf("error opening %s", filePath)
		return false
	}
	defer dir.Close()

	// Read all filenames
	contents, err := dir.Readdirnames(-1)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to read directory contents: %v", err)
		return false
	}

	for _, item := range contents {
		itemPath := filepath.Join(filePath, item)
		// Check if export server has permissions to manipulate the file
		file, err := os.Open(itemPath)
		if err != nil {
			log.Log.Reason(err).Errorf("unable to open %s, file may lack read permissions", itemPath)
			return false
		}
		file.Close()
	}
	return true
}

func checkVolumePermissions(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Log.Reason(err).Errorf("error statting %s", path)
		return false
	}
	if !fi.IsDir() {
		f, err := os.Open(path)
		if err != nil {
			log.Log.Reason(err).Errorf("error opening %s", path)
			return false
		}
		f.Close()
		return true
	}
	return checkDirectoryPermissions(path)
}

func gzipHandler(filePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f, err := os.Open(filePath)
		if err != nil {
			log.Log.Reason(err).Errorf("error opening %s", filePath)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		gzipReader := pipeToGzip(f)
		defer gzipReader.Close()
		n, err := io.Copy(w, gzipReader)
		if err != nil {
			log.Log.Reason(err).Error("error writing response body")
		}
		log.Log.Infof("Wrote %d bytes\n", n)
	})
}

func dirHandler(uri, mountPoint string) http.Handler {
	return http.StripPrefix(uri, http.FileServer(http.Dir(mountPoint)))
}

func fileHandler(file string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, "disk.img", time.Time{}, f)
	})
}

func getToken(tokenFile string) (string, error) {
	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
