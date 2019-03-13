//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"flag"
	"io/ioutil"
	"path"

	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"

	"kubevirt.io/containerized-data-importer/pkg/util"
	"kubevirt.io/containerized-data-importer/tests/utils"
)

const (
	serviceName   = "cdi-docker-registry-host"
	configMapName = serviceName + "-certs"
	certFile      = "domain.crt"
	keyFile       = "domain.key"
)

func main() {
	certDir := flag.String("certDir", "", "")
	inFile := flag.String("inFile", "", "")
	outDir := flag.String("outDir", "", "")
	flag.Parse()

	ft := &formatTable{
		[]string{""},
		[]string{".tar"},
		[]string{".tar", ".gz"},
		[]string{".qcow2"},
	}

	if err := ft.generateFiles(*inFile, *outDir); err != nil {
		glog.Fatal(errors.Wrapf(err, "generating files from %s to %s' errored: ", *inFile, *outDir))
	}

	if err := ft.populateCertDir(*certDir); err != nil {
		glog.Fatal(errors.Wrapf(err, "copy certificate directory %s' errored: ", certDir))
	}
}

func (ft formatTable) generateFiles(inFile, outDir string) error {
	glog.Info("Generating test files")
	if err := os.MkdirAll(outDir, 0777); err != nil {
		return err
	}

	if err := ft.initializeTestFiles(inFile, outDir); err != nil {
		return err
	}
	glog.Info("File initialization completed without error")

	return nil
}

func (ft formatTable) populateCertDir(certDir string) error {

	glog.Info("Creating key/certificate")

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(certDir, 0777); err != nil {
		glog.Fatal(errors.Wrapf(err, "'mkdir %s' errored: ", certDir))
	}

	namespacedName := serviceName + "." + util.GetNamespace()

	certBytes, keyBytes, err := certutil.GenerateSelfSignedCertKey(serviceName, nil, []string{namespacedName, namespacedName + ".svc"})
	if err != nil {
		return err
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
		},
		Data: map[string]string{
			certFile: string(certBytes),
		},
	}

	stored, err := clientset.CoreV1().ConfigMaps(util.GetNamespace()).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		_, err := clientset.CoreV1().ConfigMaps(util.GetNamespace()).Create(cm)
		if err != nil {
			return err
		}

	} else {
		cpy := stored.DeepCopyObject().(*v1.ConfigMap)
		cpy.Data = cm.Data
		_, err := clientset.CoreV1().ConfigMaps(util.GetNamespace()).Update(cpy)
		if err != nil {
			return err
		}
	}

	if err = ioutil.WriteFile(path.Join(certDir, certFile), certBytes, 0644); err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(certDir, keyFile), keyBytes, 0600); err != nil {
		return err
	}

	glog.Info("Successfully created key/certificate")
	return nil

}

type formatTable [][]string

func (ft formatTable) initializeTestFiles(inFile, outDir string) error {
	sem := make(chan bool, 3)
	errChan := make(chan error, len(ft))

	reportError := func(err error, msg string, format ...interface{}) {
		e := errors.Wrapf(err, msg, format...)
		glog.Error(e)
		errChan <- e
		return
	}

	for _, fList := range ft {
		sem <- true

		go func(i, o string, f []string) {
			defer func() { <-sem }()
			glog.Infof("Generating file %s\n", f)

			ext := strings.Join(f, "")
			tmpDir := filepath.Join(o, "tmp"+ext)
			if err := os.Mkdir(tmpDir, 0777); err != nil {
				reportError(err, "Error creating temp dir %s", tmpDir)
				return
			}

			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					reportError(err, "Error deleting tmp dir %s", tmpDir)
				}
			}()

			glog.Infof("Mkdir %s\n", tmpDir)

			p, err := utils.FormatTestData(i, tmpDir, f...)
			if err != nil {
				reportError(err, "Error formatting files")
				return
			}

			if err = os.Rename(p, filepath.Join(o, filepath.Base(p))); err != nil {
				reportError(err, "Error moving file %s to %s", p, o)
				return
			}

			glog.Infof("Generated file %q\n", p)
		}(inFile, outDir, fList)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
	close(errChan)

	if len(errChan) > 0 {
		for err := range errChan {
			glog.Error(err)
		}
		return errors.New("Error(s) occurred during file conversion")
	}
	return nil
}
