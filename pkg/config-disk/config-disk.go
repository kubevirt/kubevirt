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

package configdisk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
)

type ConfigDiskClient interface {
	Define(vm *v1.VM) error
	Undefine(vm *v1.VM) error
}

type configDiskClient struct {
	unixSocketPath string
	unixClient     http.Client
}

func NewConfigDiskClient(unixSocketPath string) ConfigDiskClient {
	return &configDiskClient{
		unixSocketPath: unixSocketPath,
		unixClient: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", unixSocketPath)
				},
			},
		},
	}
}

func (c *configDiskClient) Define(vm *v1.VM) error {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	if vm.Spec.CloudInit != nil {
		body, err := json.Marshal(vm.Spec.CloudInit)
		if err != nil {
			return err
		}
		var response *http.Response

		response, err = c.unixClient.Post("http://virtconfigdisk/create/cloudinit/"+namespace+"/"+domain, "application/json", strings.NewReader(string(body)))

		if err != nil {
			return err
		}

		if response.StatusCode != http.StatusOK {
			return errors.New(fmt.Sprintf("Failed to generate cloud-init info, service endpoint returned %s", response.Status))
		}
	}

	return nil
}

func (c *configDiskClient) Undefine(vm *v1.VM) error {
	if c.unixSocketPath == "" {
		return nil
	}

	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	var response *http.Response
	var err error
	response, err = c.unixClient.Post("http://virtconfigdisk/delete/all/"+namespace+"/"+domain, "", nil)

	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Failed to generate cloud-init info, service endpoint returned %s", response.Status))
	}
	return nil
}

func httpRequestHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 4 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	action := path[1]
	dataType := path[2]
	namespace := path[3]
	domain := path[4]
	body, _ := ioutil.ReadAll(r.Body)

	logging.DefaultLogger().Info().Msg(fmt.Sprintf("action %s, dataType: %s, domain %s, namespace %s", action, dataType, domain, namespace))
	switch action {
	case "create":
		switch dataType {
		case "cloudinit":
			spec := v1.CloudInitSpec{}
			err := json.Unmarshal(body, &spec)
			if err != nil {
				logging.DefaultLogger().Info().Msg(fmt.Sprintf("Failed to decode json object for domain %s in namespace %s", domain, namespace))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			err = cloudinit.GenerateLocalData(domain, namespace, &spec)
			if err != nil {
				logging.DefaultLogger().Info().Msg(fmt.Sprintf("Failed to generate local data for domain %s at namespace %s. data: %v err: %v", domain, namespace, spec, err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	case "delete":
		cloudinit.RemoveLocalData(domain, namespace)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func HttpServe(path string) {
	os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		logging.DefaultLogger().Error().Msg(err)
		panic(err)
	}
	defer listener.Close()

	http.HandleFunc("/", httpRequestHandler)
	if err := http.Serve(listener, nil); err != nil {
		logging.DefaultLogger().Error().Msg(err)
		panic(err)
	}
}
