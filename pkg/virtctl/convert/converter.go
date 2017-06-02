/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package convert

import (
	"encoding/json"
	"io"

	ghodss_yaml "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/yaml"
	kubeapi "k8s.io/client-go/pkg/api"

	virt "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

type Type string

const (
	UNSPECIFIED Type = ""
	XML         Type = "xml"
	YAML        Type = "yaml"
	JSON        Type = "json"
)

func fromYAMLOrJSON(reader io.Reader) (vm *virt.VM, err error) {
	vm = new(virt.VM)
	err = yaml.NewYAMLOrJSONDecoder(reader, 100).Decode(vm)
	return
}

func toXML(vm *virt.VM, writer io.Writer) error {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	domdesigner := designer.NewDomainDesigner(restClient, kubeapi.NamespaceDefault)
	err = domdesigner.ApplySpec(vm)
	if err != nil {
		return err
	}
	xml, err := domdesigner.Domain.Marshal()
	if err != nil {
		return err
	}
	writer.Write([]byte(xml))
	return nil
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
