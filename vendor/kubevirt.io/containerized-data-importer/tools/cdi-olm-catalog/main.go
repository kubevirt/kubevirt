package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog"
)

//CSVAnnotations - CSVAnnotations of OLM CSV manifest
type CSVAnnotations struct {
	// AlmExamples is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	AlmExamples map[string]string `json:"alm-examples,omitempty" protobuf:"bytes,12,rep,name=annotations"`
	//Capabilities
	Capabilites string `json:"capabilities,omitempty" protobuf:"bytes,1,opt,name=capabilities"`
	//Categories
	Categories string `json:"categories,omitempty" protobuf:"bytes,1,opt,name=categories"`
	//Description
	Description string `json:"description,omitempty" protobuf:"bytes,1,opt,name=description"`
}

//CSVMetadata - CSVMetadata of OLM CSV manifest
type CSVMetadata struct {
	// CSVAnnnotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	CSVAnnnotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// Name must be unique within a namespace. Is required when creating resources, although
	// some resources may allow a client to request the generation of an appropriate name
	// automatically. Name is primarily intended for creation idempotence and configuration
	// definition.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Namespace defines the space within each name must be unique. An empty namespace is
	// equivalent to the "default" namespace, but "default" is the canonical representation.
	// Not all objects are required to be scoped to a namespace - the value of this field for
	// those objects will be empty.
	//
	// Must be a DNS_LABEL.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/namespaces
	// +optional

	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
}

// ClusterServiceVersionSpec declarations tell OLM how to install an operator
// that can manage apps for a given version.
type ClusterServiceVersionSpec struct {
	InstallStrategy           olm.NamedInstallStrategy      `json:"install"`
	Version                   string                        `json:"version,omitempty"`
	Maturity                  string                        `json:"maturity,omitempty"`
	CustomResourceDefinitions olm.CustomResourceDefinitions `json:"customresourcedefinitions,omitempty"`
	APIServiceDefinitions     olm.APIServiceDefinitions     `json:"apiservicedefinitions,omitempty"`
	NativeAPIs                []metav1.GroupVersionKind     `json:"nativeAPIs,omitempty"`
	MinKubeVersion            string                        `json:"minKubeVersion,omitempty"`
	DisplayName               string                        `json:"displayName"`
	Description               string                        `json:"description,omitempty"`
	Keywords                  []string                      `json:"keywords,omitempty"`
	Maintainers               []olm.Maintainer              `json:"maintainers,omitempty"`
	Provider                  olm.AppLink                   `json:"provider,omitempty"`
	Links                     []olm.AppLink                 `json:"links,omitempty"`
	Icon                      []olm.Icon                    `json:"icon,omitempty"`

	// InstallModes specify supported installation types
	// +optional
	InstallModes []olm.InstallMode `json:"installModes,omitempty"`

	// The name of a CSV this one replaces. Should match the `metadata.Name` field of the old CSV.
	// +optional
	Replaces string `json:"replaces,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// Label selector for related resources.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`
}

//CSVManifest - struct that represents CSV manifest
type CSVManifest struct {
	APIVersion string                    `json:"apiVersion,omitempty"`
	Kind       string                    `json:"kind,omitempty"`
	CSVMetadat CSVMetadata               `json:"metadata,omitempty"`
	Spec       ClusterServiceVersionSpec `json:"spec,omitempty"`
}

//CustomResourceDefinition - custome resouirce definition
type CustomResourceDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec describes how the user wants the resources to appear
	Spec extv1beta1.CustomResourceDefinitionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

func getCSVCRDVersion(csvFile, crdKind string) (string, error) {
	var crdVersion string
	var csvStruct CSVManifest

	yamlFile, err := ioutil.ReadFile(csvFile)
	if err != nil {
		return "", errors.New("Failed to  read csv struct " + csvFile)
	}

	err = yaml.Unmarshal(yamlFile, &csvStruct)
	if err != nil {
		return "", err
	}

	crds := csvStruct.Spec.CustomResourceDefinitions.Owned
	for _, crd := range crds {
		if strings.Compare(crd.Kind, crdKind) == 0 {
			return crd.Version, nil
		} //find crd
	} //for  iterate over crds of csv

	return crdVersion, errors.New("No crd " + crdKind + " defined in csv " + csvFile)
}

func getCRDVersion(crdFile, crdKind string) (string, error) {
	var crdStruct CustomResourceDefinition

	yamlFile, err := ioutil.ReadFile(crdFile)
	if err != nil {
		return "", errors.New("Failed to  read crd manifest " + crdFile)
	}

	err = yaml.Unmarshal(yamlFile, &crdStruct)
	if err != nil {
		return "", err
	}

	//sanity
	if strings.Compare(crdKind, crdStruct.Spec.Names.Kind) != 0 {
		return "", errors.New("CRD kind " + crdStruct.Spec.Names.Kind + " does not match provided crdKind " + crdKind)
	}

	return crdStruct.Spec.Version, nil
}

const (
	help                 string = "h"
	getCrdVersionCmd     string = "get-crd-version"
	getCsvVersionCmd     string = "get-csv-crd-version"
	getCsvVersionCmdHelp string = "olm-csv-tool --cmd get-csv-crd-version --csv-file --crd-kind"
	getCrdVersionCmdHelp string = "olm-csv-tool --cmd get-crd-version --crd-file --crd-kind"
	helpInfo             string = "\nUsage:\n" + getCrdVersionCmdHelp + "\n" + getCsvVersionCmdHelp
)

func main() {

	cmd := flag.String("cmd", "h", "command")
	csvFile := flag.String("csv-file", "", "csv manifest")
	crdFile := flag.String("crd-file", "", "crd manifest")
	crdKind := flag.String("crd-kind", "CDI", "name of crd")
	flag.Parse()

	switch *cmd {
	case getCrdVersionCmd:
		if *crdFile == "" {
			klog.Errorf(getCrdVersionCmdHelp)
			klog.Error("No crd manifest provided on command %v", getCrdVersionCmd)
			panic(nil)
		}

		if *crdKind == "" {
			klog.Errorf(getCrdVersionCmdHelp)
			klog.Error("No crd-kind provided on command %v", getCrdVersionCmd)
			panic(nil)
		}

		crdVersion, err := getCRDVersion(*crdFile, *crdKind)
		if err != nil {
			panic(err)
		}
		fmt.Println(crdVersion)

		break
	case getCsvVersionCmd:
		if *csvFile == "" {
			klog.Errorf(getCsvVersionCmdHelp)
			klog.Error("No csv manifest provided on command %v", getCsvVersionCmd)
			panic(nil)
		}
		if *crdKind == "" {
			klog.Errorf(getCsvVersionCmdHelp)
			klog.Error("No crd-kind provided on command %v", getCsvVersionCmd)
			panic(nil)
		}

		crdVersion, err := getCSVCRDVersion(*csvFile, *crdKind)
		if err != nil {
			panic(err)
		}

		fmt.Println(crdVersion)

		break
	case help:
		fmt.Println(helpInfo)
		break
	default:
		panic("Invalid command " + *cmd + helpInfo)
	}
}
