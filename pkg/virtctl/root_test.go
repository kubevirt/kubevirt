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
 * Copyright 2018 Red Hat, Inc.
 * Copyright 2018 The Kubernetes Authors.
 *
 */

package virtctl

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

var exampleYaml = `name: virt
shortdesc: kubevirt command plugin
longdesc: ""
example: ""
command: ./virtctl
flags: []
tree:
- name: console
  shortdesc: Connect to a console of a virtual machine.
  longdesc: ""
  example: ""
  command: ./virtctl console
  flags:
  - name: timeout
    shorthand: ""
    desc: The number of minutes to wait for the virtual machine instance to be ready.
    defvalue: "5"
  tree: []
- name: expose
  shortdesc: Expose a virtual machine as a new service.
  longdesc: ""
  example: ""
  command: ./virtctl expose
  flags:
  - name: cluster-ip
    shorthand: ""
    desc: ClusterIP to be assigned to the service. Leave empty to auto-allocate, or
      set to 'None' to create a headless service.
    defvalue: ""
  - name: external-ip
    shorthand: ""
    desc: Additional external IP address (not managed by the cluster) to accept for
      the service. If this IP is routed to a node, the service can be accessed by
      this IP in addition to its generated service IP. Optional.
    defvalue: ""
  - name: load-balancer-ip
    shorthand: ""
    desc: IP to assign to the Load Balancer. If empty, an ephemeral IP will be created
      and used.
    defvalue: ""
  - name: name
    shorthand: ""
    desc: Name of the service created for the exposure of the VM
    defvalue: ""
  - name: node-port
    shorthand: ""
    desc: Port used to expose the service on each node in a cluster.
    defvalue: "0"
  - name: port
    shorthand: ""
    desc: The port that the service should serve on
    defvalue: "0"
  - name: port-name
    shorthand: ""
    desc: Name of the port. Optional.
    defvalue: ""
  - name: protocol
    shorthand: ""
    desc: The network protocol for the service to be created.
    defvalue: TCP
  - name: target-port
    shorthand: ""
    desc: Name or number for the port on the VM that the service should direct traffic
      to. Optional.
    defvalue: ""
  - name: type
    shorthand: ""
    desc: 'Type for this service: ClusterIP, NodePort, or LoadBalancer.'
    defvalue: ClusterIP
  tree: []
- name: start
  shortdesc: Start a virtual machine instance which is managed by a virtual machine.
  longdesc: ""
  example: ""
  command: ./virtctl start
  flags: []
  tree: []
- name: stop
  shortdesc: Stop a virtual machine instace which is managed by a virtual machine.
  longdesc: ""
  example: ""
  command: ./virtctl stop
  flags: []
  tree: []
- name: version
  shortdesc: Print the client and server version information
  longdesc: ""
  example: ""
  command: ./virtctl version
  flags: []
  tree: []
- name: vnc
  shortdesc: Open a vnc connection to a virtual machine.
  longdesc: ""
  example: ""
  command: ./virtctl vnc
  flags: []
  tree: []
`

var _ = Describe("Kubevirt Root Client", func() {
	var command *cobra.Command
	var plugin *kubecli.Plugin
	var workDir string

	BeforeEach(func() {
		workDir, err := ioutil.TempDir("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())
		command = NewVirtctlCommand()
		plugin = kubecli.MakePluginConfiguration(workDir, command)
	})

	AfterEach(func() {
		os.RemoveAll(workDir)
	})
	Context("With example yaml check install command", func() {
		It("Marshal struct into yaml", func() {
			yamlData, err := yaml.Marshal(plugin)
			Expect(err).ToNot(HaveOccurred())
			Expect(exampleYaml).To(Equal(string(yamlData)))
		})
	})
})
