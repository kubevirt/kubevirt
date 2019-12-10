package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/mfranczy/crd-rest-coverage/pkg/report"
)

func main() {
	var (
		auditLogPath   string
		swaggerPath    string
		outputJSONPath string
		detailed       bool
	)

	// TODO: add filter param
	flag.StringVar(&swaggerPath, "swagger-path", "", "path to swagger file")
	flag.StringVar(&auditLogPath, "audit-log-path", "", "path to k8s audit log file")
	flag.StringVar(&outputJSONPath, "output-path", "", "destination path for report file")
	flag.BoolVar(&detailed, "detailed", false, "show report with coverage for each endpoint")
	flag.Parse()

	// TODO: improve glog format
	if swaggerPath == "" || auditLogPath == "" {
		glog.Exitf("params --swagger-path and --audit-log-path are required")
	}

	coverage, err := report.Generate(auditLogPath, swaggerPath, "")
	if err != nil {
		glog.Exit(err)
	}

	if outputJSONPath != "" {
		report.Dump(outputJSONPath, coverage)
	} else {
		report.Print(coverage, detailed)
	}
}
