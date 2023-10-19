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
	"crypto/tls"
	"encoding/json"
	"errors"
	goflag "flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"kubevirt.io/client-go/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/service"
)

const (
	authHeader              = "x-kubevirt-export-token"
	manifestCmBasePath      = "/manifest_data/"
	vmManifestPath          = manifestCmBasePath + "virtualmachine-manifest"
	internalLinkPath        = manifestCmBasePath + "internal_host"
	internalCaConfigMapPath = manifestCmBasePath + "internal_ca_cm"
	externalLinkPath        = manifestCmBasePath + "external_host"
	externalCaConfigMapPath = manifestCmBasePath + "external_ca_cm"
	exportNamePath          = manifestCmBasePath + "export-name"

	external = "/external"
	internal = "/internal"
)

type TokenGetterFunc func() (string, error)

type VolumeInfo struct {
	Path       string
	ArchiveURI string
	DirURI     string
	RawURI     string
	RawGzURI   string
	VMURI      string
	SecretURI  string
}
type ExportServerConfig struct {
	Deadline time.Time

	ListenAddr string

	CertFile, KeyFile string

	TokenFile string

	Volumes []VolumeInfo

	// unit testing helpers
	ArchiveHandler     func(string) http.Handler
	DirHandler         func(string, string) http.Handler
	FileHandler        func(string) http.Handler
	GzipHandler        func(string) http.Handler
	VmHandler          func(string, []VolumeInfo, func() (string, error), func() (*corev1.ConfigMap, error)) http.Handler
	TokenSecretHandler func(TokenGetterFunc) http.Handler

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
			errBytes, _ := io.ReadAll(er.stderr)
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
	for i, vi := range s.Volumes {
		for path, handler := range s.getHandlerMap(vi) {
			log.Log.Infof("Handling path %s\n", path)
			mux.Handle(path, tokenChecker(s.TokenGetter, handler))
		}
		if i == 0 {
			// Only register once
			if vi.VMURI != "" {
				p := vi.Path
				mux.Handle(filepath.Join(internal, vi.VMURI), tokenChecker(s.TokenGetter, s.VmHandler(p, s.Volumes, getInternalBasePath, getInternalCAConfigMap)))
				mux.Handle(filepath.Join(external, vi.VMURI), tokenChecker(s.TokenGetter, s.VmHandler(p, s.Volumes, getExternalBasePath, getExternalCAConfigMap)))
			}
			if vi.SecretURI != "" {
				mux.Handle(filepath.Join(internal, vi.SecretURI), tokenChecker(s.TokenGetter, s.TokenSecretHandler(s.TokenGetter)))
				mux.Handle(filepath.Join(external, vi.SecretURI), tokenChecker(s.TokenGetter, s.TokenSecretHandler(s.TokenGetter)))
			}
		}
	}

	s.handler = mux
}

func getInternalCAConfigMap() (*corev1.ConfigMap, error) {
	return getCAConfigMap(internalCaConfigMapPath)
}

func getExternalCAConfigMap() (*corev1.ConfigMap, error) {
	return getCAConfigMap(externalCaConfigMapPath)
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
		// Disable HTTP/2
		// See CVE-2023-44487
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
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

	if es.VmHandler == nil {
		es.VmHandler = vmHandler
	}

	if es.TokenSecretHandler == nil {
		es.TokenSecretHandler = secretHandler
	}

	if es.TokenGetter == nil {
		es.TokenGetter = func() (string, error) {
			return getToken(es.TokenFile)
		}
	}

	return es
}

var getExpandedVM = func() *virtv1.VirtualMachine {
	f, err := os.Open(vmManifestPath)
	if err != nil {
		log.Log.Reason(err).Info("Unable to load VM manifest data")
		return nil
	}
	defer f.Close()
	fileinfo, err := f.Stat()
	if err != nil {
		log.Log.Reason(err).Info("Unable to load VM manifest data")
		return nil
	}
	buf := make([]byte, fileinfo.Size())
	_, err = f.Read(buf)
	if err != nil {
		log.Log.Reason(err).Info("Unable to load VM manifest data")
		return nil
	}

	vm := &virtv1.VirtualMachine{}
	if err := json.Unmarshal(buf, vm); err != nil {
		log.Log.Reason(err).Info("Unable to load VM manifest data")
		return nil
	}
	return vm
}

var getInternalBasePath = func() (string, error) {
	data, err := os.ReadFile(internalLinkPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

var getExportName = func() (string, error) {
	data, err := os.ReadFile(exportNamePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

var getExternalBasePath = func() (string, error) {
	data, err := os.ReadFile(externalLinkPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetTypeMetaString(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("apiVersion: %s\nkind: %s\n", gvk.GroupVersion().String(), gvk.Kind)
}

var getCAConfigMap = func(name string) (*corev1.ConfigMap, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileinfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, fileinfo.Size())
	_, err = f.Read(buf)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{}
	if err := json.Unmarshal(buf, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

var getCdiHeaderSecret = func(token, name string) *corev1.Secret {
	data := make(map[string]string)

	data["token"] = fmt.Sprintf("x-kubevirt-export-token:%s", token)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: data,
	}
}

var getDataVolumes = func(vm *virtv1.VirtualMachine) ([]*cdiv1.DataVolume, error) {
	res := make([]*cdiv1.DataVolume, 0)
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		name := ""
		if volume.DataVolume != nil {
			name = volume.DataVolume.Name
		} else if volume.PersistentVolumeClaim != nil {
			name = volume.PersistentVolumeClaim.ClaimName
		}
		if name == "" {
			continue
		}
		log.Log.V(1).Infof("Opening DV %s", filepath.Join(manifestCmBasePath, fmt.Sprintf("dv-%s", name)))
		f, err := os.Open(filepath.Join(manifestCmBasePath, fmt.Sprintf("dv-%s", name)))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Log.V(1).Info("DV not found skipping")
				continue
			}
			return nil, err
		}
		defer f.Close()
		fileinfo, err := f.Stat()
		if err != nil {
			return nil, err
		}
		buf := make([]byte, fileinfo.Size())
		_, err = f.Read(buf)
		if err != nil {
			return nil, err
		}
		dv := &cdiv1.DataVolume{}
		if err := json.Unmarshal(buf, dv); err != nil {
			return nil, err
		}
		res = append(res, dv)
	}
	return res, nil
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

	return &execReader{cmd: cmd, stdout: stdout, stderr: io.NopCloser(&stderr)}, nil
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

func vmHandler(filePath string, vi []VolumeInfo, getBasePath func() (string, error), getCmFunc func() (*corev1.ConfigMap, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resources := make([]runtime.Object, 0)
		outputFunc := resourceToBytesJson
		contentType := req.Header.Get("Accept")
		if contentType == runtime.ContentTypeYAML {
			outputFunc = resourceToBytesYaml
		}
		exportName, err := getExportName()
		if err != nil {
			log.Log.Reason(err).Error("error reading export name")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		headerSecretName := getSecretTokenName(exportName)
		path, err := getBasePath()
		if err != nil {
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					log.Log.Reason(err).Info("path not found")
					w.WriteHeader(http.StatusNotFound)
				} else {
					log.Log.Reason(err).Error("error reading path")
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
			return
		}
		certCm, error := getCmFunc()
		if error != nil {
			log.Log.Reason(err).Error("error reading ca configmap information")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		certCm.TypeMeta = metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		}
		resources = append(resources, certCm)
		expandedVm := getExpandedVM()
		if expandedVm == nil {
			log.Log.Reason(err).Error("error getting VM definition")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		expandedVm.TypeMeta = metav1.TypeMeta{
			Kind:       virtv1.VirtualMachineGroupVersionKind.Kind,
			APIVersion: virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		}
		for i, dvTemplate := range expandedVm.Spec.DataVolumeTemplates {
			dvTemplate.Spec.Source.HTTP.URL = fmt.Sprintf("https://%s", filepath.Join(path, vi[i].RawGzURI))
			dvTemplate.Spec.Source.HTTP.CertConfigMap = certCm.Name
			dvTemplate.Spec.Source.HTTP.SecretExtraHeaders = []string{headerSecretName}
		}
		resources = append(resources, expandedVm)
		datavolumes, err := getDataVolumes(expandedVm)
		if err != nil {
			log.Log.Reason(err).Error("error reading datavolumes information")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, dv := range datavolumes {
			dv.TypeMeta = metav1.TypeMeta{
				Kind:       "DataVolume",
				APIVersion: "cdi.kubevirt.io/v1beta1",
			}
			for _, info := range vi {
				if strings.Contains(info.RawGzURI, dv.Name) {
					dv.Spec.Source.HTTP.URL = fmt.Sprintf("https://%s", filepath.Join(path, info.RawGzURI))
				}
			}
			dv.Spec.Source.HTTP.CertConfigMap = certCm.Name
			dv.Spec.Source.HTTP.SecretExtraHeaders = []string{headerSecretName}
			resources = append(resources, dv)
		}
		data, err := outputFunc(resources)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		n, err := w.Write(data)
		if err != nil {
			log.Log.Reason(err).Error("error writing manifests")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Log.Infof("Wrote %d bytes\n", n)
	})
}

func resourceToBytesJson(resources []runtime.Object) ([]byte, error) {
	list := corev1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}
	for _, resource := range resources {
		list.Items = append(list.Items, runtime.RawExtension{Object: resource})
	}
	resourceBytes, err := json.MarshalIndent(list, "", "    ")
	if err != nil {
		return nil, err
	}
	return resourceBytes, nil
}

func resourceToBytesYaml(resources []runtime.Object) ([]byte, error) {
	data := []byte{}
	for _, resource := range resources {
		resourceBytes, err := yaml.Marshal(resource)
		if err != nil {
			return nil, err
		}
		data = append(data, resourceBytes...)
		data = append(data, []byte("---\n")...)
	}
	return data, nil
}

func dirHandler(uri, mountPoint string) http.Handler {
	return http.StripPrefix(uri, http.FileServer(http.Dir(mountPoint)))
}

func fileHandler(file string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(file)
		if err != nil {
			log.Log.Reason(err).Errorf("error opening %s", file)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, "disk.img", time.Time{}, f)
	})
}

func getToken(tokenFile string) (string, error) {
	content, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

var getSecretTokenName = func(exportName string) string {
	return fmt.Sprintf("header-secret-%s", exportName)
}

func secretHandler(tokenGetter TokenGetterFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resources := make([]runtime.Object, 0)
		outputFunc := resourceToBytesJson
		contentType := req.Header.Get("Accept")
		if contentType == runtime.ContentTypeYAML {
			outputFunc = resourceToBytesYaml
		}
		token, err := tokenGetter()
		if err != nil {
			log.Log.Reason(err).Error("error getting token")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		exportName, err := getExportName()
		if err != nil {
			log.Log.Reason(err).Error("error reading export name")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		headerSecretName := getSecretTokenName(exportName)
		secret := getCdiHeaderSecret(token, headerSecretName)
		secret.TypeMeta = metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		}
		resources = append(resources, secret)
		data, err := outputFunc(resources)
		if err != nil {
			log.Log.Reason(err).Errorf("error generating secret manifest")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		n, err := w.Write(data)
		if err != nil {
			log.Log.Reason(err).Error("error writing secret manifest")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Log.Infof("Wrote %d bytes\n", n)
	})
}
