package util

import hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

type CsvWithComponent struct {
	Name      string
	Csv       string
	Component hcoutil.AppComponent
}
