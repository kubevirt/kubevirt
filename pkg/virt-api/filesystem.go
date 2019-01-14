package virt_api

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE -imports =os

import (
	"io"
	"io/ioutil"
	"os"
)

type File interface {
	io.Closer
	io.Writer
}

type Filesystem interface {
	WriteFile(filename string, data []byte, perm os.FileMode) error
	TempDir(dir, prefix string) (name string, err error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
}

type IOUtil struct{}

func (IOUtil) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (IOUtil) TempDir(dir, prefix string) (name string, err error) {
	return ioutil.TempDir(dir, prefix)
}

func (IOUtil) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}
