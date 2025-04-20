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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
)

var mdevBasePath string = "/sys/bus/mdev/devices"
var mdevClassBusPath string = "/sys/class/mdev_bus"

func createMDEVType(mdevType string, parentID string, uid string) error {

	path := filepath.Join(mdevClassBusPath, parentID, "mdev_supported_types", mdevType, "create")
	// wait for interface to become available
	if !isInterfaceAvailable(path) {
		msg := fmt.Sprintf("failed to create mdev type %s, interface is not available %s", mdevType, path)
		errMsg := fmt.Errorf(msg)
		fmt.Println(msg)
		return errMsg
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0200)
	if err != nil {
		fmt.Printf("failed to create mdev type %s, can't open path %s\n", mdevType, path)
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(uid); err != nil {
		fmt.Printf("failed to create mdev type %s, can't write to %s\n", mdevType, path)
		return err
	}
	fmt.Printf("Successfully created mdev %s - %s\n", mdevType, uid)
	return nil
}

func removeMDEVType(mdevUUID string) error {
	removePath := filepath.Join(mdevBasePath, mdevUUID, "remove")
	// wait for interface to become available
	if !isInterfaceAvailable(removePath) {
		msg := fmt.Sprintf("failed to remove mdev %s, interface is not available %s", mdevUUID, removePath)
		errMsg := fmt.Errorf(msg)
		fmt.Println(msg)
		return errMsg
	}

	f, err := os.OpenFile(removePath, os.O_WRONLY, 0200)
	if err != nil {
		fmt.Printf("failed to remove mdev %s, can't open path %s\n", mdevUUID, removePath)
		return err
	}

	defer f.Close()

	if _, err = f.WriteString("1"); err != nil {
		fmt.Printf("failed to remove mdev %s, can't write to %s\n", mdevUUID, removePath)
		return err
	}
	fmt.Printf("Successfully removed mdev %s\n", mdevUUID)
	return nil
}

func NewCreateMDEVCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create-mdev",
		Short: "create a mediate device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			mdevType := cmd.Flag("type").Value.String()
			parentID := cmd.Flag("parent").Value.String()
			UID := cmd.Flag("uuid").Value.String()
			return createMDEVType(mdevType, parentID, UID)
		},
	}
}

func NewRemoveMDEVCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-mdev",
		Short: "remove a mediate device",
		RunE: func(cmd *cobra.Command, args []string) error {
			mdevUUID := cmd.Flag("uuid").Value.String()
			return removeMDEVType(mdevUUID)
		},
	}
}

func isInterfaceAvailable(interfacePath string) bool {
	err := virtwait.PollImmediately(1*time.Second, 5*time.Second, func(_ context.Context) (done bool, err error) {
		_, err = os.Stat(interfacePath)
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		fmt.Printf("interface %s is not available after multiple tries\n", interfacePath)
		return false
	}
	return true
}
