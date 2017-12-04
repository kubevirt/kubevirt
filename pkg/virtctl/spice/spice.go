/*
 * This file is part of the KubeVirt project
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

package spice

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	flag "github.com/spf13/pflag"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"gopkg.in/ini.v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

const FLAG = "spice"
const TEMP_PREFIX = "spice"

type Spice struct {
}

func DownloadSpice(namespace string, vm string, restClient *rest.RESTClient) (*v1.Spice, error) {
	spice := &v1.Spice{}
	err := restClient.Get().
		Resource("virtualmachines").SetHeader("Accept", "application/json").
		SubResource("spice").
		Namespace(namespace).
		Name(vm).Do().Into(spice)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Can't fetch connection details: %s\n", err.Error()))
	}
	return spice, nil
}

func (o *Spice) FlagSet() *flag.FlagSet {

	cf := flag.NewFlagSet(FLAG, flag.ExitOnError)
	cf.BoolP("details", "d", false, "If present, print SPICE console to stdout, otherwise run remote-viewer")
	cf.StringP("proxy", "p", "", "If given, will override any given proxy from the server")
	return cf
}

func (o *Spice) Run(flags *flag.FlagSet) int {
	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	details, _ := flags.GetBool("details")
	proxy, _ := flags.GetString("proxy")
	namespace, _ := flags.GetString("namespace")
	if namespace == "" {
		namespace = kubev1.NamespaceDefault
	}

	if len(flags.Args()) != 2 {
		log.Println("VM name is missing")
		return 1
	}
	vm := flags.Arg(1)

	virtClient, err := kubecli.GetKubevirtClientFromFlags(server, kubeconfig)

	if err != nil {
		log.Println(err)
		return 1
	}
	spice, err := DownloadSpice(namespace, vm, virtClient.RestClient())
	if err != nil {
		log.Fatalf(err.Error())
		return 1
	}
	if proxy != "" {
		spice.Info.Proxy = proxy
	}
	cfg := ini.Empty()
	err = ini.ReflectFrom(cfg, spice)
	if err != nil {
		log.Fatalf("Can't serialize spice struct to ini")
		return 1
	}
	if details {
		_, err := cfg.WriteTo(os.Stdout)
		if err != nil {
			log.Fatalf("Failed to write to stdout")
			return 1
		}
	} else {
		f, err := ioutil.TempFile("", TEMP_PREFIX)

		if err != nil {
			log.Fatalf("Can't open file: %s", err.Error())
			return 1
		}
		defer os.Remove(f.Name())
		defer f.Close()

		_, err = cfg.WriteTo(f)
		if err != nil {
			log.Fatalf("Can't write to file: %s", err.Error())
			return 1
		}

		f.Sync()

		cmnd := exec.Command("remote-viewer", f.Name())
		err = cmnd.Run()

		if err != nil {
			log.Fatalf("Something goes wring with remote-viewer: %s", err.Error())
			return 1
		}
	}
	return 0
}

func (o *Spice) Usage() string {
	usage := "virtctl can connect via remote-viewer to a VM, or can show SPICE connection details\n\n"
	usage += "Examples:\n"
	usage += "# Show SPICE connection details of the VM testvm\n"
	usage += "./virtctl spice testvm --details\n\n"
	usage += "# Connect to testvm via remote-viewer\n"
	usage += "./virtctl spice testvm\n\n"
	usage += "# Connect to testvm via remote-viewer using a proxy\n"
	usage += "./virtctl spice testvm --proxy http://192.168.200.2:1234\n\n"
	usage += "Options:\n"
	usage += o.FlagSet().FlagUsages()
	return usage
}
