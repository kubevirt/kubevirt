package main

// importer.go implements a data fetching service capable of pulling objects from remote object
// stores and writing to a local directory. It utilizes the minio-go client sdk for s3 remotes,
// https for public remotes, and "file" for local files. The main use-case for this importer is
// to copy VM images to a "golden" namespace for consumption by kubevirt.
// This process expects several environmental variables:
//    ImporterEndpoint       Endpoint url minus scheme, bucket/object and port, eg. s3.amazon.com.
//			      Access and secret keys are optional. If omitted no creds are passed
//			      to the object store client.
//    ImporterAccessKeyID  Optional. Access key is the user ID that uniquely identifies your
//			      account.
//    ImporterSecretKey     Optional. Secret key is the password to your account.

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/resource"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/pkg/image"
	"kubevirt.io/containerized-data-importer/pkg/importer"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

func init() {
	flag.Parse()
}

func main() {
	defer glog.Flush()

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)
	util.StartPrometheusEndpoint(certsDirectory)

	glog.V(1).Infoln("Starting importer")
	ep, _ := util.ParseEnvVar(common.ImporterEndpoint, false)
	acc, _ := util.ParseEnvVar(common.ImporterAccessKeyID, false)
	sec, _ := util.ParseEnvVar(common.ImporterSecretKey, false)
	source, _ := util.ParseEnvVar(common.ImporterSource, false)
	contentType, _ := util.ParseEnvVar(common.ImporterContentType, false)
	imageSize, _ := util.ParseEnvVar(common.ImporterImageSize, false)

	dest := common.ImporterWritePath
	if contentType == controller.ContentTypeArchive || source == controller.SourceRegistry {
		dest = common.ImporterVolumePath
	}

	glog.V(1).Infoln("begin import process")
	dso := &importer.DataStreamOptions{
		dest,
		ep,
		acc,
		sec,
		source,
		contentType,
		imageSize,
	}

	if source == controller.SourceNone && contentType == controller.ContentTypeKubevirt {
		requestImageSizeQuantity := resource.MustParse(imageSize)
		minSizeQuantity := util.MinQuantity(resource.NewScaledQuantity(util.GetAvailableSpace(common.ImporterVolumePath), 0), &requestImageSizeQuantity)
		if minSizeQuantity.Cmp(requestImageSizeQuantity) != 0 {
			// Available dest space is smaller than the size we want to create
			glog.Warningf("Available space less than requested size, creating blank image sized to available space: %s.\n", minSizeQuantity.String())
		}
		err := image.CreateBlankImage(common.ImporterWritePath, minSizeQuantity)
		if err != nil {
			glog.Errorf("%+v", err)
			os.Exit(1)
		}
	} else {
		glog.V(1).Infoln("begin import process")
		err = importer.CopyData(dso)
		if err != nil {
			glog.Errorf("%+v", err)
			os.Exit(1)
		}
	}
	glog.V(1).Infoln("import complete")
}
