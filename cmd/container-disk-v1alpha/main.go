package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kubevirt.io/kubevirt/pkg/log"
)

const (
	EnvCopyPath            = "COPY_PATH"
	EnvImagePath           = "IMAGE_PATH"
	DiskSourceFallbackPath = "/disk"
	QEMUIMGPath            = "/usr/bin/qemu-img"
	ReadinessProbeFile     = "/tmp/healthy"
)

type DiskInfo struct {
	Format      string `json:"format"`
	BackingFile string `json:"backing-filename"`
}

func main() {
	var err error

	logger := log.DefaultLogger()

	copyPath := os.Getenv(EnvCopyPath)
	if copyPath == "" {
		logger.Errorf("Environment Variable %s is not defined.", EnvCopyPath)
		os.Exit(1)
	}

	imagePath := os.Getenv(EnvImagePath)
	imagePath, err = GetImage(imagePath)
	if err != nil {
		logger.Reason(err).Errorf("Image lookup failed.")
		os.Exit(1)
	}

	imageType, err := VerifyImage(imagePath)
	if err != nil {
		logger.Reason(err).Error("Could not determine image type.")
		os.Exit(1)
	}

	err = CopyImage(imagePath, copyPath, imageType)
	if err != nil {
		logger.Reason(err).Errorf("Failed to copy image to the destination.")
		os.Exit(1)
	}

	f, err := os.Create(ReadinessProbeFile)
	if err != nil {
		logger.Reason(err).Errorf("Failed to mark myself as ready.")
		os.Exit(1)
	}
	f.Close()

	for {
		time.Sleep(1 * time.Second)
	}
}

func GetImage(imagePath string) (string, error) {
	if imagePath != "" {
		if _, err := os.Stat(imagePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("No image on path %s", imagePath)
			} else {
				return "", fmt.Errorf("Failed to check if an image exists at %s", imagePath)
			}
		}
	} else {
		files, err := ioutil.ReadDir(DiskSourceFallbackPath)
		if err != nil {
			return "", fmt.Errorf("Failed to check %s for disks: %v", DiskSourceFallbackPath, err)
		}
		if len(files) > 1 {
			return "", fmt.Errorf("More than one file found in folder %s, only one disk is allowed", DiskSourceFallbackPath)
		}
		imagePath = filepath.Join(DiskSourceFallbackPath, files[0].Name())
	}
	return imagePath, nil
}

func GetImageInfo(imagePath string) (*DiskInfo, error) {
	out, err := exec.Command(QEMUIMGPath, "info", imagePath, "--output", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to invoke qemu-img: %v", err)
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}

func VerifyQCOW2(diskInfo *DiskInfo) error {
	if diskInfo.Format != "qcow2" {
		return fmt.Errorf("expected a disk format of qcow2, but got '%v'", diskInfo.Format)
	}

	if diskInfo.BackingFile != "" {
		return fmt.Errorf("expected no backing file, but found %v", diskInfo.BackingFile)
	}
	return nil
}

func VerifyImage(imagePath string) (string, error) {
	if diskInfo, err := GetImageInfo(imagePath); err != nil {
		return "", err
	} else {
		switch diskInfo.Format {
		case "qcow2":
			return diskInfo.Format, VerifyQCOW2(diskInfo)
		case "raw":
			return diskInfo.Format, nil
		default:
			return diskInfo.Format, fmt.Errorf("unsupported image format: %v", diskInfo.Format)
		}
	}
}

func CopyImage(imagePath string, targetPath string, imageType string) error {
	err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to target path directory: %v", err)
	}

	switch imageType {
	case "raw":
		return Copy(imagePath, targetPath, imageType)
	case "qcow2":
		return ConvertAndCopy(imagePath, targetPath)
	default:
		return fmt.Errorf("image format %s not supported", imageType)
	}
}

func ConvertAndCopy(imagePath string, targetPath string) error {
	out, err := exec.Command(QEMUIMGPath, "convert", imagePath, fmt.Sprintf("%s.%s", targetPath, "raw")).CombinedOutput()
	if err != nil {
		log.DefaultLogger().Warningf("image conversion failed with output: %s", string(out))
		return err
	}
	return nil
}

func Copy(imagePath string, targetPath string, imageType string) error {
	targetfile := fmt.Sprintf("%s.%s", targetPath, imageType)
	target, err := os.Create(targetfile)
	if err != nil {
		return fmt.Errorf("failed to crate target image file: %v", err)
	}
	defer target.Close()
	source, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open source image file: %v", err)
	}
	defer source.Close()
	_, err = io.Copy(target, source)
	if err != nil {
		return fmt.Errorf("failed to copy image: %v", err)
	}
	return nil
}
