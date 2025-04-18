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
	"sync/atomic"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type Libvirt struct {
	VirtConnection *virtConnection
	VirtDomain     *virtDomain
	stackCounter   *atomic.Int32
}

func NewLibvirt(ctrl *gomock.Controller) *Libvirt {
	counter := &atomic.Int32{}
	mockLibvirt := &Libvirt{
		newVirtConnection(ctrl, counter),
		newVirtDomain(ctrl, counter),
		counter,
	}
	ginkgo.DeferCleanup(func() {
		gomega.Expect(mockLibvirt.StackCount()).To(gomega.BeZero(), "You are introducing a leak. A Domain resource was not freed.")
	})
	return mockLibvirt
}

func (l *Libvirt) StackCount() int32 {
	return l.stackCounter.Load()
}

func (l *Libvirt) ConnectionEXPECT() *cli.MockConnectionMockRecorder {
	return l.VirtConnection.EXPECT()
}

func (l *Libvirt) DomainEXPECT() *cli.MockVirDomainMockRecorder {
	return l.VirtDomain.EXPECT()
}

type virtDomain struct {
	*cli.MockVirDomain
	*atomic.Int32
}

func newVirtDomain(ctrl *gomock.Controller, counter *atomic.Int32) *virtDomain {
	return &virtDomain{
		cli.NewMockVirDomain(ctrl),
		counter,
	}
}

func (d *virtDomain) Free() error {
	if d.Load() > 0 {
		d.Add(int32(-1))
	}
	return d.MockVirDomain.Free()
}

type virtConnection struct {
	*cli.MockConnection
	*atomic.Int32
}

func newVirtConnection(ctrl *gomock.Controller, counter *atomic.Int32) *virtConnection {
	return &virtConnection{
		cli.NewMockConnection(ctrl),
		counter,
	}
}

func (c *virtConnection) LookupDomainByName(name string) (cli.VirDomain, error) {
	val, err := c.MockConnection.LookupDomainByName(name)
	if err != nil {
		return nil, err
	}
	c.Add(1)
	return val, nil
}

func (c *virtConnection) DomainDefineXML(xml string) (cli.VirDomain, error) {
	val, err := c.MockConnection.DomainDefineXML(xml)
	if err != nil {
		return nil, err
	}
	c.Add(1)
	return val, nil
}

func (c *virtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]cli.VirDomain, error) {
	val, err := c.MockConnection.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	c.Add(int32(len(val)))
	return val, nil
}

func (c *virtConnection) GetAllDomainStats(statsTypes libvirt.DomainStatsTypes, flags libvirt.ConnectGetAllDomainStatsFlags) ([]libvirt.DomainStats, error) {
	val, err := c.MockConnection.GetAllDomainStats(statsTypes, flags)
	if err != nil {
		return nil, err
	}
	c.Add(int32(len(val)))
	return val, nil
}
