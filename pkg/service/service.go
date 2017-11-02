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
	"fmt"
	"strconv"
)

type Service interface {
	Run()
}

type ServiceListen struct {
	Name string
	Host string
	Port string
}

func NewServiceListen(name string, host *string, port *int) *ServiceListen {
	return &ServiceListen{
		Name: name,
		Host: host,
		Port: strconv.Itoa(port),
	}
}

func (service *ServiceListen) Address() string {
	return fmt.Sprintf("%s:%s", service.Host, service.Port)
}
