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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

func main() {
	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")

	socket := pflag.String("socket", cmdclient.SocketOnGuest(), "Socket for connecting to the cmd server")
	domainName := pflag.String("domainName", "", "Domain Name of the Virtual Machine to connect the agent to. Usually namespace_vmname")
	command := pflag.String("command", "", "Command to execute on the guest")
	memProfile := pflag.String("memProfile", "", "Path to store a memory profile. Profiling is skipped if empty")
	timeoutSeconds := pflag.Int32("timeoutSeconds", 1, "Duration in seconds the probe will wait for the guest command to return.")
	guestAgentPing := pflag.Bool("guestAgentPing", false, "Flag to specify readiness probe based of guest-agent ping")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.Parse()

	log.InitializeLogging("virt-probe")

	client, err := cmdclient.NewClient(*socket)
	if err != nil {
		log.Log.Reason(err).Error("Failed to connect cmd client")
		os.Exit(1)
	}

	if *guestAgentPing {
		err := client.GuestPing(*domainName, *timeoutSeconds)
		if err != nil {
			log.Log.Reason(err).Critical("Failed to ping the guest")
			os.Exit(1)
		}
		os.Exit(0)
	}

	exitCode, stdOut, err := client.Exec(*domainName, *command, pflag.Args(), *timeoutSeconds)
	if len(stdOut) > 0 {
		fmt.Println(stdOut)
	}
	if err != nil {
		log.Log.Reason(err).Critical("Failed executing the command")
	}

	saveMemoryProfile(*memProfile)
	os.Exit(exitCode)
}

func saveMemoryProfile(path string) {
	if len(path) > 0 {
		log.Log.Info("creating memory profile")
		f, err := os.Create(path)
		if err != nil {
			log.Log.Reason(err).Critical("creating memory profile")
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Log.Reason(err).Critical("writing memory profile")
		}
		f.Close()
		return
	}
}
