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
 * Copyright The KubeVirt Authors.
 */

package main

import (
	"errors"
	goflag "flag"

	"github.com/golang/mock/gomock"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
)

func main() {
	socket := pflag.String("socket", cmdclient.SocketOnGuest(), "Socket for the cmd server")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.Parse()

	log.InitializeLogging("fake-cmd-server")

	stopChan := make(chan struct{})
	options := cmdserver.NewServerOptions(true)

	domainManager := virtwrap.NewMockDomainManager(gomock.NewController(nil))
	domainManager.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().DoAndReturn(func(domainName string, _ string, _ []string) (string, error) {
		if domainName == "error" {
			return "", errors.New("fake error")
		}
		if domainName == "fail" {
			return "command failed", agent.ExecExitCode{ExitCode: 1}
		}
		return "success", nil
	})
	log.Log.Info("running fake server")
	done, err := cmdserver.RunServer(*socket, domainManager, stopChan, options)
	if err != nil {
		log.Log.Reason(err).Critical("running cmd server")
	}

	<-done
}
