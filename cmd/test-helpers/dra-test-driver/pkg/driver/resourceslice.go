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
 *
 */

package driver

import (
	"context"
	"log"

	resourceapi "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const maxDevices = 5

func PublishResourceSlice(ctx context.Context, clientset *kubernetes.Clientset, nodeName, driverName string) error {
	devices := []resourceapi.Device{}
	for i := 0; i < maxDevices; i++ {
		devices = append(devices, resourceapi.Device{
			Name:       "hostpath-" + string(rune('0'+i)),
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{},
			Capacity:   map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{},
		})
	}

	slice := &resourceapi.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName + "-" + driverName,
		},
		Spec: resourceapi.ResourceSliceSpec{
			NodeName: &nodeName,
			Driver:   driverName,
			Pool:     resourceapi.ResourcePool{Name: "hostpath", ResourceSliceCount: 1},
			Devices:  devices,
		},
	}

	_, err := clientset.ResourceV1().ResourceSlices().Create(ctx, slice, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create ResourceSlice: %v", err)
		return err
	}
	log.Printf("Published ResourceSlice with %d devices", maxDevices)
	return nil
}
