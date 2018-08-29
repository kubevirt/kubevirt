package main

// importer.go implements a data fetching service capable of pulling objects from remote object
// stores and writing to a local directory. It utilizes the minio-go client sdk for s3 remotes,
// https for public remotes, and "file" for local files. The main use-case for this importer is
// to copy VM images to a "golden" namespace for consumption by kubevirt.
// This process expects several environmental variables:
//    IMPORTER_ENDPOINT       Endpoint url minus scheme, bucket/object and port, eg. s3.amazon.com.
//			      Access and secret keys are optional. If omitted no creds are passed
//			      to the object store client.
//    IMPORTER_ACCESS_KEY_ID  Optional. Access key is the user ID that uniquely identifies your
//			      account.
//    IMPORTER_SECRET_KEY     Optional. Secret key is the password to your account.

import (
	"flag"
	"os"

	"github.com/golang/glog"
	. "kubevirt.io/containerized-data-importer/pkg/common"
	. "kubevirt.io/containerized-data-importer/pkg/importer"
)

func init() {
	flag.Parse()
}

func main() {
	defer glog.Flush()

	glog.V(Vuser).Infoln("Starting importer")
	ep, _ := ParseEnvVar(IMPORTER_ENDPOINT, false)
	acc, _ := ParseEnvVar(IMPORTER_ACCESS_KEY_ID, false)
	sec, _ := ParseEnvVar(IMPORTER_SECRET_KEY, false)

	glog.V(Vuser).Infoln("begin import process")
	err := CopyImage(IMPORTER_WRITE_PATH, ep, acc, sec)
	if err != nil {
		glog.Errorf("%+v", err)
		os.Exit(1)
	}
	glog.V(Vuser).Infoln("import complete")
}
