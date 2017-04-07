package convert

import (
	"encoding/json"
	"encoding/xml"
	ghodss_yaml "github.com/ghodss/yaml"
	"github.com/jeevatkm/go-model"
	"io"
	"k8s.io/client-go/pkg/util/yaml"
	virt "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

type Type string

const (
	UNSPECIFIED Type = ""
	XML         Type = "xml"
	YAML        Type = "yaml"
	JSON        Type = "json"
)

func fromXML(reader io.Reader) (vm *virt.VM, err error) {
	domainSpec := api.DomainSpec{}
	err = xml.NewDecoder(reader).Decode(&domainSpec)
	if err != nil {
		return
	}
	vm = virt.NewMinimalVM(domainSpec.Name)
	if e := model.Copy(vm.Spec.Domain, &domainSpec); len(e) > 0 {
		err = e[0]
		return
	}
	return
}

func fromYAMLOrJSON(reader io.Reader) (vm *virt.VM, err error) {
	vm = new(virt.VM)
	err = yaml.NewYAMLOrJSONDecoder(reader, 100).Decode(vm)
	return
}

func toXML(vm *virt.VM, writer io.Writer) error {
	domainSpec := new(api.DomainSpec)
	if e := model.Copy(domainSpec, vm.Spec.Domain); len(e) > 0 {
		return e[0]
	}
	domainSpec.Name = vm.GetObjectMeta().GetName()
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	return encoder.Encode(domainSpec)
}

func toYAML(vm *virt.VM, writer io.Writer) error {
	b, err := json.Marshal(vm)
	if err != nil {
		return err
	}
	b, err = ghodss_yaml.JSONToYAML(b)
	if err != nil {
		return err
	}
	_, err = writer.Write(b)
	return err
}

func toJSON(vm *virt.VM, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(vm)
}

type Encoder func(*virt.VM, io.Writer) error
type Decoder func(io.Reader) (*virt.VM, error)
