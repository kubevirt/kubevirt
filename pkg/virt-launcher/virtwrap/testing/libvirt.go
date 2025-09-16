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

package testing

import (
	"fmt"
	"sync"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type callStack struct {
	stack map[cli.VirDomain]uint32
	mutex sync.Mutex
}

type Libvirt struct {
	VirtConnection *virtConnection
	VirtDomain     *virtDomain
	VirtStream     *virtStream
	callStack      *callStack
}

func NewLibvirt(ctrl *gomock.Controller) *Libvirt {
	cs := &callStack{
		stack: make(map[cli.VirDomain]uint32),
	}
	mockLibvirt := &Libvirt{
		newVirtConnection(ctrl, cs),
		newVirtDomain(ctrl, cs),
		newVirtStream(ctrl, cs),
		cs,
	}
	ginkgo.DeferCleanup(func() {
		gomega.ExpectWithOffset(1, mockLibvirt.callStackEmpty()).To(gomega.Succeed(), "You are introducing a leak. A Domain resource was not freed.")
	})
	return mockLibvirt
}

func (l *Libvirt) callStackEmpty() error {
	l.callStack.mutex.Lock()
	defer l.callStack.mutex.Unlock()
	pendingDomainCounter := 0
	for _, val := range l.callStack.stack {
		if val != uint32(0) {
			pendingDomainCounter++
		}
	}
	if pendingDomainCounter != 0 {
		return fmt.Errorf("there are %d domain requests that are not freed", pendingDomainCounter)
	}
	return nil
}

func (l *Libvirt) ConnectionEXPECT() *cli.MockConnectionMockRecorder {
	return l.VirtConnection.EXPECT()
}

func (l *Libvirt) DomainEXPECT() *cli.MockVirDomainMockRecorder {
	return l.VirtDomain.EXPECT()
}

type virtStream struct {
	*cli.MockStream
	callStack *callStack
}

func newVirtStream(ctrl *gomock.Controller, callStack *callStack) *virtStream {
	return &virtStream{
		cli.NewMockStream(ctrl),
		callStack,
	}
}

type virtDomain struct {
	*cli.MockVirDomain
	callStack *callStack
}

func newVirtDomain(ctrl *gomock.Controller, callStack *callStack) *virtDomain {
	return &virtDomain{
		cli.NewMockVirDomain(ctrl),
		callStack,
	}
}

func (d *virtDomain) Free() error {
	d.callStack.mutex.Lock()
	defer d.callStack.mutex.Unlock()
	if d.callStack.stack[d] > uint32(0) {
		d.callStack.stack[d]--
	}
	return d.MockVirDomain.Free()
}

type virtConnection struct {
	*cli.MockConnection
	callStack *callStack
}

func newVirtConnection(ctrl *gomock.Controller, callStack *callStack) *virtConnection {
	return &virtConnection{
		cli.NewMockConnection(ctrl),
		callStack,
	}
}

func (c *virtConnection) LookupDomainByName(name string) (cli.VirDomain, error) {
	val, err := c.MockConnection.LookupDomainByName(name)
	if err != nil {
		return nil, err
	}
	c.callStack.mutex.Lock()
	defer c.callStack.mutex.Unlock()
	c.callStack.stack[val]++
	return val, nil
}

func (c *virtConnection) DomainDefineXML(xml string) (cli.VirDomain, error) {
	val, err := c.MockConnection.DomainDefineXML(xml)
	if err != nil {
		return nil, err
	}
	c.callStack.mutex.Lock()
	defer c.callStack.mutex.Unlock()
	c.callStack.stack[val]++
	return val, nil
}

func (c *virtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]cli.VirDomain, error) {
	val, err := c.MockConnection.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	c.callStack.mutex.Lock()
	defer c.callStack.mutex.Unlock()
	for _, domain := range val {
		c.callStack.stack[domain]++
	}
	return val, nil
}

func (c *virtConnection) GetAllDomainStats(statsTypes libvirt.DomainStatsTypes, flags libvirt.ConnectGetAllDomainStatsFlags) ([]libvirt.DomainStats, error) {
	val, err := c.MockConnection.GetAllDomainStats(statsTypes, flags)
	if err != nil {
		return nil, err
	}
	c.callStack.mutex.Lock()
	defer c.callStack.mutex.Unlock()
	for _, domainStats := range val {
		c.callStack.stack[domainStats.Domain]++
	}
	return val, nil
}
