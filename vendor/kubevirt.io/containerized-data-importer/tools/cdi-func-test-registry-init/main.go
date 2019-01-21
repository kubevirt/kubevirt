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

	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"kubevirt.io/containerized-data-importer/tests/utils"
)

func main() {
	inCertDir := flag.String("inCertDir", "", "")
	outCertDir := flag.String("outCertDir", "", "")
	inFile := flag.String("inFile", "", "")
	outDir := flag.String("outDir", "", "")
	flag.Parse()

	ft := &formatTable{
		[]string{""},
		[]string{".tar"},
		[]string{".gz"},
		[]string{".xz"},
		[]string{".tar", ".gz"},
		[]string{".tar", ".xz"},
		[]string{".qcow2"},
	}

	if err := ft.generateFiles(*inFile, *outDir); err != nil {
		glog.Fatal(errors.Wrapf(err, "generating files from %s to %s' errored: ", *inFile, *outDir))
	}

	if err := ft.copyCertDir(*inCertDir, *outCertDir); err != nil {
		glog.Fatal(errors.Wrapf(err, "copy certificate directory %s' errored: ", outCertDir))
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

func (ft formatTable) copyFile(src, dest string, info os.FileInfo) error {

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

func (ft formatTable) copyCertDir(inCertDir string, outCertDir string) error {

	glog.Info("Copying certificates")

	if err := os.MkdirAll(outCertDir, 0777); err != nil {
		glog.Fatal(errors.Wrapf(err, "'mkdir %s' errored: ", outCertDir))
	}

	contents, err := ioutil.ReadDir(inCertDir)
	if err != nil {
		return err
	}

	for _, content := range contents {
		cs, cd := filepath.Join(inCertDir, content.Name()), filepath.Join(outCertDir, content.Name())
		if err := ft.copyFile(cs, cd, content); err != nil {
			// If any error exit
			return err
		}
	}

	glog.Info("Copying certificates completed without errors")
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
