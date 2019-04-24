package util

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/common"
)

// CountingReader is a reader that keeps track of how much has been read
type CountingReader struct {
	Reader  io.ReadCloser
	Current int64
}

// RandAlphaNum provides an implementation to generate a random alpha numeric string of the specified length
func RandAlphaNum(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

// GetNamespace returns the namespace the pod is executing in
func GetNamespace() string {
	return getNamespace("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
}

func getNamespace(path string) string {
	if data, err := ioutil.ReadFile(path); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "cdi"
}

// ParseEnvVar provides a wrapper to attempt to fetch the specified env var
func ParseEnvVar(envVarName string, decode bool) (string, error) {
	value := os.Getenv(envVarName)
	if decode {
		v, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", errors.Errorf("error decoding environment variable %q", envVarName)
		}
		value = fmt.Sprintf("%s", v)
	}
	return value, nil
}

// Read reads bytes from the stream and updates the prometheus clone_progress metric according to the progress.
func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Current += int64(n)
	return n, err
}

// Close closes the stream
func (r *CountingReader) Close() error {
	return r.Reader.Close()
}

// GetAvailableSpaceByVolumeMode calls another method based on the volumeMode parameter to get the amount of
// available space at the path specified.
func GetAvailableSpaceByVolumeMode(volumeMode v1.PersistentVolumeMode) int64 {
	if volumeMode == v1.PersistentVolumeBlock {
		return GetAvailableSpaceBlock(common.ImporterWriteBlockPath)
	}
	return GetAvailableSpace(common.ImporterVolumePath)
}

// GetAvailableSpace gets the amount of available space at the path specified.
func GetAvailableSpace(path string) int64 {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return int64(-1)
	}
	return int64(stat.Bavail) * int64(stat.Bsize)
}

// GetAvailableSpaceBlock gets the amount of available space at the block device path specified.
func GetAvailableSpaceBlock(deviceName string) int64 {
	cmd := exec.Command("/usr/bin/lsblk", "-n", "-b", "-o", "SIZE", deviceName)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return int64(-1)
	}
	i, err := strconv.ParseInt(strings.TrimSpace(string(out.Bytes())), 10, 64)
	if err != nil {
		return int64(-1)
	}
	return i
}

// MinQuantity calculates the minimum of two quantities.
func MinQuantity(availableSpace, imageSize *resource.Quantity) resource.Quantity {
	if imageSize.Cmp(*availableSpace) == 1 {
		return *availableSpace
	}
	return *imageSize
}

// UnArchiveTar unarchives a tar file and streams its files
// using the specified io.Reader to the specified destination.
func UnArchiveTar(reader io.Reader, destDir string, arg ...string) error {
	klog.V(1).Infof("begin untar...\n")

	var tarOptions string
	var args = arg
	if len(arg) > 0 {
		tarOptions = arg[0]
		args = arg[1:]
	}
	options := fmt.Sprintf("-%s%s", tarOptions, "xvC")
	untar := exec.Command("/usr/bin/tar", options, destDir, strings.Join(args, ""))
	untar.Stdin = reader
	var errBuf bytes.Buffer
	untar.Stderr = &errBuf
	err := untar.Start()
	if err != nil {
		return err
	}
	err = untar.Wait()
	if err != nil {
		klog.V(3).Infof("%s\n", string(errBuf.Bytes()))
		klog.Errorf("%s\n", err.Error())
		return err
	}
	return nil
}

// UnArchiveLocalTar unarchives a local tar file to the specified destination.
func UnArchiveLocalTar(filePath, destDir string, arg ...string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "could not open tar file")
	}
	fileReader := bufio.NewReader(file)
	return UnArchiveTar(fileReader, destDir, arg...)
}

// CopyFile copies a file from one location to another.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
