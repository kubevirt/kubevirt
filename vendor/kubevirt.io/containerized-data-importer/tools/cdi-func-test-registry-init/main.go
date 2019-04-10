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

	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog"

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
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})

	ft := &formatTable{
		[]string{""},
		[]string{".tar"},
		[]string{".tar", ".gz"},
		[]string{".qcow2"},
	}

	if err := ft.generateFiles(*inFile, *outDir); err != nil {
		klog.Fatal(errors.Wrapf(err, "generating files from %s to %s' errored: ", *inFile, *outDir))
	}

	if err := utils.CreateCertForTestService(util.GetNamespace(), serviceName, configMapName, *certDir, certFile, keyFile); err != nil {
		klog.Fatal(errors.Wrapf(err, "populate certificate directory %s' errored: ", *certDir))
	}
}

func (ft formatTable) generateFiles(inFile, outDir string) error {
	klog.Info("Generating test files")
	if err := os.MkdirAll(outDir, 0777); err != nil {
		return err
	}

	if err := ft.initializeTestFiles(inFile, outDir); err != nil {
		return err
	}
	klog.Info("File initialization completed without error")

	return nil
}

type formatTable [][]string

func (ft formatTable) initializeTestFiles(inFile, outDir string) error {
	sem := make(chan bool, 2)
	errChan := make(chan error, len(ft))

	reportError := func(err error, msg string, format ...interface{}) {
		e := errors.Wrapf(err, msg, format...)
		klog.Error(e)
		errChan <- e
		return
	}

	for _, fList := range ft {
		sem <- true

		go func(i, o string, f []string) {
			defer func() { <-sem }()
			klog.Infof("Generating file %s\n", f)

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

			klog.Infof("Mkdir %s\n", tmpDir)

			p, err := utils.FormatTestData(i, tmpDir, f...)
			if err != nil {
				reportError(err, "Error formatting files")
				return
			}

			if err = os.Rename(p, filepath.Join(o, filepath.Base(p))); err != nil {
				reportError(err, "Error moving file %s to %s", p, o)
				return
			}

			klog.Infof("Generated file %q\n", p)
		}(inFile, outDir, fList)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
	close(errChan)

	if len(errChan) > 0 {
		for err := range errChan {
			klog.Error(err)
		}
		return errors.New("Error(s) occurred during file conversion")
	}
	return nil
}
