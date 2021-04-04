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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"os/user"

	"github.com/spf13/cobra"
	"github.com/syndtr/gocapability/capability"
)

var (
	envinfo bool
)

func addEnvInfoPersistentFlag(command *cobra.Command) {
	command.PersistentFlags().BoolVar(&envinfo, "envinfo", false, "Report environment information")
}

func executeEnvInfo() error {
	if !envinfo {
		return nil
	}

	if err := reportUser(); err != nil {
		return err
	}
	if err := reportCapabilities(0); err != nil {
		return err
	}
	return nil
}

func reportCapabilities(pid int32) error {
	caps, err := capability.NewPid2(0)
	if err != nil {
		return fmt.Errorf("unable to report capabilities: %v", err)
	}

	if err := caps.Load(); err != nil {
		return fmt.Errorf("unable to report capabilities: %v", err)
	}

	fmt.Printf("Capabilities:\n%s\n\n", caps)

	return nil
}

func reportUser() error {
	selfUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to report user information: %v", err)
	}
	fmt.Printf("\nUser: %s (%s/%s)\n", selfUser.Username, selfUser.Uid, selfUser.Gid)

	return nil
}
