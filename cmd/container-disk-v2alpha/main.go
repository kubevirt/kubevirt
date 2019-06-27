package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"kubevirt.io/client-go/log"
)

const (
	ReadinessProbeFile = "/healthy"
)

func main() {
	var err error

	var copyPath string
	var healthCheck bool

	logger := log.DefaultLogger()

	flag.StringVar(&copyPath, "copy-path", "", "Image target path")
	flag.BoolVar(&healthCheck, "health-check", false, "Do a health check")
	flag.Parse()

	if !healthCheck && copyPath == "" {
		logger.Error("No copy-path provides.")
		os.Exit(1)
	}

	if healthCheck {
		_, err := os.Stat(ReadinessProbeFile)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	err = os.MkdirAll(filepath.Dir(copyPath), os.ModePerm)
	if err != nil {
		logger.Reason(err).Errorf("Failed to create disk directory %s.", filepath.Dir(copyPath))
		os.Exit(1)
	}

	ln, err := net.Listen("unix", fmt.Sprintf("%s.%s", copyPath, "sock"))
	if err != nil {
		logger.Reason(err).Error("Failed to create socket.")
		os.Exit(1)
	}
	defer ln.Close()

	f, err := os.Create(ReadinessProbeFile)
	if err != nil {
		logger.Reason(err).Errorf("Failed to mark myself as ready.")
		os.Exit(1)
	}
	f.Close()

	for {
		_, err := ln.Accept()
		if err != nil {
			logger.Reason(err).Error("Unrecoverable error on socket.")
			os.Exit(1)
		}
	}
}
