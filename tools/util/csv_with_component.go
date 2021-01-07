package util

import hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

type CsvWithComponent struct {
	Csv       string
	Component hcoutil.AppComponent
}
