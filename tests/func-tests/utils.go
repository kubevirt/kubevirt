package tests

import (
	"os"
)

const (
	testNamespace  = "kubevirt-hyperconverged"
	vmiName        = "testvmi"
	kubevirtCfgMap = "kubevirt-config"
)

//GetJobTypeEnvVar returns "JOB_TYPE" enviroment varibale
func GetJobTypeEnvVar() string {
	return (os.Getenv("JOB_TYPE"))
}

// PanicOnError raises the given err if exist
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
