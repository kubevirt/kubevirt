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

package virtcontroller

const (
	testNamespace          = "test-ns"
	testNodeName           = "testNode"
	testVMIName            = "testvmi"
	testVMINameDashed      = "test-vmi"
	testVMName             = "testvm"
	testVMNameDashed       = "test-vm"
	testSecondVMNameDashed = "test-vm-2"
	testVMIMName           = "testvmimigration"

	testAnnotationOSCentos8      = "centos8"
	testAnnotationOSCentos7      = "centos7"
	testAnnotationWorkloadServer = "server"
	testAnnotationFlavorTiny     = "tiny"
	testAnnotationFlavorMedium   = "medium"
	testAnnotationDummy          = "dummy"
	testConditionAny             = "any"
	testPhaseRunningLower        = "running"
	testVMIPhaseRunning          = "Running"
	testVMIPhaseScheduling       = "Scheduling"

	testMigrationName = "test-migration"
	testMigrationUID  = "test-migration-uid"
	testDataVolumePVC = "test-dv-pvc"
	testRootDiskName  = "rootdisk"
	testVMLocalAlias  = "vm1"
	testVNICIface1    = "iface1"
	testVNICIface2    = "iface2"
	testVNICIface3    = "iface3"
	testVNICIface4    = "iface4"
	testMultusNetwork = "multus-net"

	testLabelEnvironment     = "environment"
	testLabelTeam            = "team"
	testLabelVersion         = "version"
	testLabelValueProduction = "production"

	testInfoSourceGuestAgent = "guest-agent"
	testBridgeInterfaceName  = "br-int"
	testVMIInterfaceName     = "net-0"
	testVMIInterfaceBridge   = "br-ex"
	testVMIInterfaceIP       = "10.11.126.126"
	testOVSSystemInterface   = "ovs-system"
	testModelE1000E          = "e1000e"
	testModelVirtio          = "virtio"
	testNetworkCustom        = "custom-net"
	testCustomPluginName     = "custom-plugin"
)
