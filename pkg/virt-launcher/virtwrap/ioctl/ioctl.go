package ioctl

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"os"

	"golang.org/x/sys/unix"
)

type Helper interface {
	IoctlGetInt(req uint) (int, error)
	Close() error
}

type realHelper struct {
	file *os.File
}

func (r *realHelper) IoctlGetInt(req uint) (int, error) {
	return unix.IoctlGetInt(int(r.file.Fd()), req)
}

func (r *realHelper) Close() error {
	return r.file.Close()
}

func New(path string) (Helper, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &realHelper{file: file}, nil
}

type Factory interface {
	New(path string) (Helper, error)
}

type realFactory struct{}

func (realFactory) New(path string) (Helper, error) {
	return New(path)
}

func NewFactory() Factory {
	return realFactory{}
}
