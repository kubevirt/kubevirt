package schema

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	apiservervalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func readPragmas(fileBytes []byte) (pragmas []string, err error) {
	fileReader := bytes.NewReader(fileBytes)
	fileBufReader := bufio.NewReader(fileReader)
	for {
		maybePragma, err := fileBufReader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(maybePragma, "#!") {
			pragmas = append(pragmas, strings.TrimSpace(strings.TrimPrefix(maybePragma, "#!")))
		} else {
			// pragmas must be defined at the top of the file, stop when we don't see a line with the pragma mark
			break
		}
	}
	return
}

type Meta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
}

func (m *Meta) GetObjectKind() schema.ObjectKind {
	return m
}
func (in *Meta) DeepCopyInto(out *Meta) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	return
}

func (in *Meta) DeepCopy() *Meta {
	if in == nil {
		return nil
	}
	out := new(Meta)
	in.DeepCopyInto(out)
	return out
}

func (in *Meta) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

func validateKubectlable(fileBytes []byte) error {
	exampleFileBytesJson, err := yaml.YAMLToJSON(fileBytes)
	if err != nil {
		return err
	}
	parsedMeta := &Meta{}
	err = json.Unmarshal(exampleFileBytesJson, parsedMeta)
	if err != nil {
		return err
	}
	requiresNamespace := parsedMeta.Kind != "CustomResourceDefinition"
	errs := apiValidation.ValidateObjectMeta(
		&parsedMeta.ObjectMeta,
		requiresNamespace,
		func(s string, prefix bool) []string {
			return nil
		},
		field.NewPath("metadata"),
	)

	if len(errs) > 0 {
		return fmt.Errorf("error validating object metadata: %s. %v. %s", errs, parsedMeta, string(exampleFileBytesJson))
	}
	return nil
}

func validateUsingPragma(pragma string, fileBytes []byte) (bool, error) {
	const validateCRDPrefix = "validate-crd:"
	const ParseAsKindPrefix = "parse-kind:"
	const PackageManifest = "package-manifest:"

	switch {
	case strings.HasPrefix(pragma, validateCRDPrefix):
		return true, validateCRD(strings.TrimSpace(strings.TrimPrefix(pragma, validateCRDPrefix)), fileBytes)
	case strings.HasPrefix(pragma, ParseAsKindPrefix):
		return true, validateKind(strings.TrimSpace(strings.TrimPrefix(pragma, ParseAsKindPrefix)), fileBytes)
	case strings.HasPrefix(pragma, PackageManifest):
		csvFilenames := strings.Split(strings.TrimSpace(strings.TrimPrefix(pragma, PackageManifest)), ",")
		return false, validatePackageManifest(fileBytes, csvFilenames)
	}
	return false, nil
}

func validatePackageManifest(fileBytes []byte, csvFilenames []string) error {
	manifestBytesJson, err := yaml.YAMLToJSON(fileBytes)
	if err != nil {
		return err
	}

	var packageManifest registry.PackageManifest
	err = json.Unmarshal(manifestBytesJson, &packageManifest)
	if err != nil {
		return err
	}

	if len(packageManifest.Channels) < 1 {
		return fmt.Errorf("Package manifest validation failure for package %s: Missing channels", packageManifest.PackageName)
	}

	// Collect the defined CSV names.
	csvNames := map[string]bool{}
	for _, csvFilename := range csvFilenames {
		csvBytes, err := ioutil.ReadFile(csvFilename)
		if err != nil {
			return err
		}

		csvBytesJson, err := yaml.YAMLToJSON(csvBytes)
		if err != nil {
			return err
		}

		csv := v1alpha1.ClusterServiceVersion{}
		err = json.Unmarshal(csvBytesJson, &csv)
		if err != nil {
			return err
		}

		csvNames[csv.Name] = true
	}

	if len(packageManifest.PackageName) == 0 {
		return fmt.Errorf("Empty package name")
	}

	// Make sure that each channel name is unique and that the referenced CSV exists.
	channelMap := make(map[string]bool, len(packageManifest.Channels))
	for _, channel := range packageManifest.Channels {
		if _, exists := channelMap[channel.Name]; exists {
			return fmt.Errorf("Channel %s declared twice in package manifest", channel.Name)
		}

		if _, ok := csvNames[channel.CurrentCSVName]; !ok {
			return fmt.Errorf("Missing CSV with name %s", channel.CurrentCSVName)
		}

		channelMap[channel.Name] = true
	}

	return nil
}

func validateCRD(schemaFileName string, fileBytes []byte) error {
	schemaBytes, err := ioutil.ReadFile(schemaFileName)
	if err != nil {
		return err
	}
	schemaBytesJson, err := yaml.YAMLToJSON(schemaBytes)
	if err != nil {
		return err
	}

	crd := v1beta1.CustomResourceDefinition{}
	json.Unmarshal(schemaBytesJson, &crd)

	exampleFileBytesJson, err := yaml.YAMLToJSON(fileBytes)
	if err != nil {
		return err
	}
	unstructured := unstructured.Unstructured{}
	err = json.Unmarshal(exampleFileBytesJson, &unstructured)
	if err != nil {
		return err
	}

	// Validate CRD definition statically
	scheme := runtime.NewScheme()
	err = apiextensions.AddToScheme(scheme)
	if err != nil {
		return err
	}
	err = v1beta1.AddToScheme(scheme)
	if err != nil {
		return err
	}

	unversionedCRD := apiextensions.CustomResourceDefinition{}
	scheme.Converter().Convert(&crd, &unversionedCRD, conversion.SourceToDest, nil)
	errList := validation.ValidateCustomResourceDefinition(&unversionedCRD)
	if len(errList) > 0 {
		for _, ferr := range errList {
			fmt.Println(ferr)
		}
		return fmt.Errorf("CRD failed validation: %s. Errors: %s", schemaFileName, errList)
	}

	// Validate CR against CRD schema
	validator, _, err := apiservervalidation.NewSchemaValidator(unversionedCRD.Spec.Validation)
	return apiservervalidation.ValidateCustomResource(unstructured.UnstructuredContent(), validator)
}

func validateKind(kind string, fileBytes []byte) error {
	exampleFileBytesJson, err := yaml.YAMLToJSON(fileBytes)
	if err != nil {
		return err
	}

	switch kind {
	case "ClusterServiceVersion":
		csv := v1alpha1.ClusterServiceVersion{}
		err = json.Unmarshal(exampleFileBytesJson, &csv)
		if err != nil {
			return err
		}
		return err
	case "CatalogSource":
		cs := v1alpha1.CatalogSource{}
		err = json.Unmarshal(exampleFileBytesJson, &cs)
		if err != nil {
			return err
		}
		return err
	default:
		return fmt.Errorf("didn't recognize validate-kind directive: %s", kind)
	}
}

func validateResource(path string, f os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	exampleFileReader, err := os.Open(path)
	if err != nil {
		return err
	}
	defer exampleFileReader.Close()

	fileReader := bufio.NewReader(exampleFileReader)
	fileBytes, err := ioutil.ReadAll(fileReader)
	if err != nil {
		return err
	}
	pragmas, err := readPragmas(fileBytes)
	if err != nil {
		return err
	}

	isKubResource := false
	for _, pragma := range pragmas {
		fileReader.Reset(exampleFileReader)
		isKub, err := validateUsingPragma(pragma, fileBytes)
		if err != nil {
			return fmt.Errorf("validating %s: %v", path, err)
		}
		isKubResource = isKubResource || isKub
	}

	if isKubResource {
		err = validateKubectlable(fileBytes)
		if err != nil {
			return fmt.Errorf("validating %s: %v", path, err)
		}
	}
	return nil
}

func validateResources(directory string) error {
	err := filepath.Walk(directory, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		fmt.Printf("validate %s\n", path)
		if validateResource(path, f, err) != nil {
			return err
		}

		return nil
	})
	return err
}

func CheckCatalogResources(manifestDir string) error {
	return validateResources(manifestDir)
}
