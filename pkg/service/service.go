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

package service

import (
	goflag "flag"
	"fmt"
	"strconv"

	flag "github.com/spf13/pflag"
)

type Service interface {
	Run()
	AddFlags()
}

type ServiceListen struct {
	Name        string
	BindAddress string
	Port        int
}

type ServiceLibvirt struct {
	LibvirtUri string
}

func (service *ServiceListen) Address() string {
	return fmt.Sprintf("%s:%s", service.BindAddress, strconv.Itoa(service.Port))
}

func (service *ServiceListen) InitFlags() {
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("kubeconfig"))
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("master"))
}

func (service *ServiceListen) AddCommonFlags() {
	flag.StringVar(&service.BindAddress, "listen", service.BindAddress, "Address where to listen on")
	flag.IntVar(&service.Port, "port", service.Port, "Port to listen on")
}

func (service *ServiceLibvirt) AddLibvirtFlags() {
	flag.StringVar(&service.LibvirtUri, "libvirt-uri", service.LibvirtUri, "Libvirt connection string")

}

func Setup(service Service) {
	service.AddFlags()

	// set new default verbosity, was set to 0 by glog
	flag.Set("v", "2")

	flag.Parse()
}
