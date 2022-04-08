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

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"kubevirt.io/client-go/log"
)

const (
	authHeader = "x-kubevirt-export-token"
	port       = 8443
)

type execReader struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

type volumeInfo struct {
	path       string
	archiveURI string
	dirURI     string
	rawURI     string
	rawGzURI   string
}

func (vi volumeInfo) getHandlers() map[string]http.Handler {
	var result = make(map[string]http.Handler)
	if vi.archiveURI != "" {
		result[vi.archiveURI] = archiveHandler(vi.path)
	}
	if vi.dirURI != "" {
		result[vi.dirURI] = dirHandler(vi.dirURI, vi.path)
	}
	if vi.rawURI != "" {
		fi, err := os.Stat(vi.path)
		if err != nil {
			log.Log.Reason(err).Errorf("error statting %s", vi.path)
		} else {
			if fi.IsDir() {
				result[vi.rawURI] = fileHandler(path.Join(vi.path, "disk.img"))
			} else {
				result[vi.rawURI] = fileHandler(vi.path)
			}
		}
	}
	if vi.rawGzURI != "" {
		fi, err := os.Stat(vi.path)
		if err != nil {
			log.Log.Reason(err).Errorf("error statting %s", vi.path)
		} else {
			if fi.IsDir() {
				result[vi.rawGzURI] = gzipHandler(path.Join(vi.path, "disk.img"))
			} else {
				result[vi.rawGzURI] = gzipHandler(vi.path)
			}
		}
	}
	return result
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

func tokenChecker(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := getToken()
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

func gzipHandler(filePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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

func getCert() (certFile, keyFile string) {
	certFile = os.Getenv("CERT_FILE")
	keyFile = os.Getenv("KEY_FILE")
	if certFile == "" || keyFile == "" {
		panic("TLS config incomplete")
	}
	return
}

func getToken() (string, error) {
	tokenFile := os.Getenv("TOKEN_FILE")
	if tokenFile == "" {
		return "", fmt.Errorf("no token file set")
	}

	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func getVolumeInfo() []volumeInfo {
	var result []volumeInfo
	for _, env := range os.Environ() {
		kv := strings.Split(env, "=")
		envPrefix := strings.TrimSuffix(kv[0], "_EXPORT_PATH")
		if envPrefix != kv[0] {
			vi := volumeInfo{
				path:       kv[1],
				archiveURI: os.Getenv(envPrefix + "_EXPORT_ARCHIVE_URI"),
				dirURI:     os.Getenv(envPrefix + "_EXPORT_DIR_URI"),
				rawURI:     os.Getenv(envPrefix + "_EXPORT_RAW_URI"),
				rawGzURI:   os.Getenv(envPrefix + "_EXPORT_RAW_GZIP_URI"),
			}
			result = append(result, vi)
		}
	}
	return result
}

func main() {
	log.InitializeLogging("virt-exportserver-" + os.Getenv("POD_NAME"))
	log.Log.Info("Starting export server")

	certFile, keyFile := getCert()
	volumeInfo := getVolumeInfo()

	mux := http.NewServeMux()
	for _, vi := range volumeInfo {
		for path, handler := range vi.getHandlers() {
			log.Log.Infof("Handling path %s\n", path)
			mux.Handle(path, tokenChecker(handler))
		}
	}

	srv := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}

	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
		panic(err)
	}
}
