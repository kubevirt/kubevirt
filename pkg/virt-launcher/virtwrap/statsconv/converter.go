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
 *
 */

package statsconv

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/libvirt/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"github.com/libvirt/libvirt-go"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type DomainIdentifier interface {
	GetName() (string, error)
	GetUUIDString() (string, error)
}

func Convert_libvirt_DomainStats_to_stats_DomainStats(ident DomainIdentifier, in *libvirt.DomainStats, inMem []libvirt.DomainMemoryStat, out *stats.DomainStats) error {
	name, err := ident.GetName()
	if err != nil {
		return err
	}
	out.Name = name

	uuid, err := ident.GetUUIDString()
	if err != nil {
		return err
	}
	out.UUID = uuid

	out.Cpu = Convert_libvirt_DomainStatsCpu_To_stats_DomainStatsCpu(in.Cpu)
	out.Memory = Convert_libvirt_MemoryStat_to_stats_DomainStatsMemory(inMem)
	out.Vcpu = Convert_libvirt_DomainStatsVcpu_To_stats_DomainStatsVcpu(in.Vcpu)
	out.Net = Convert_libvirt_DomainStatsNet_To_stats_DomainStatsNet(in.Net)
	out.Block = Convert_libvirt_DomainStatsBlock_To_stats_DomainStatsBlock(in.Block)

	return nil
}

func Convert_libvirt_DomainStatsCpu_To_stats_DomainStatsCpu(in *libvirt.DomainStatsCPU) *stats.DomainStatsCPU {
	if in == nil {
		return &stats.DomainStatsCPU{}
	}

	return &stats.DomainStatsCPU{
		TimeSet:   in.TimeSet,
		Time:      in.Time,
		UserSet:   in.UserSet,
		User:      in.User,
		SystemSet: in.SystemSet,
		System:    in.System,
	}
}

func Convert_libvirt_MemoryStat_to_stats_DomainStatsMemory(inMem []libvirt.DomainMemoryStat) *stats.DomainStatsMemory {
	ret := &stats.DomainStatsMemory{}
	for _, stat := range inMem {
		tag := libvirt.DomainMemoryStatTags(stat.Tag)
		switch tag {
		case libvirt.DOMAIN_MEMORY_STAT_UNUSED:
			ret.UnusedSet = true
			ret.Unused = stat.Val
		case libvirt.DOMAIN_MEMORY_STAT_AVAILABLE:
			ret.AvailableSet = true
			ret.Available = stat.Val
		case libvirt.DOMAIN_MEMORY_STAT_ACTUAL_BALLOON:
			ret.ActualBalloonSet = true
			ret.ActualBalloon = stat.Val
		case libvirt.DOMAIN_MEMORY_STAT_RSS:
			ret.RSSSet = true
			ret.RSS = stat.Val
		}
	}
	return ret
}

func Convert_libvirt_DomainStatsVcpu_To_stats_DomainStatsVcpu(in []libvirt.DomainStatsVcpu) []stats.DomainStatsVcpu {
	ret := make([]stats.DomainStatsVcpu, 0, len(in))
	for _, inItem := range in {
		ret = append(ret, stats.DomainStatsVcpu{
			StateSet: inItem.StateSet,
			State:    int(inItem.State),
			TimeSet:  inItem.TimeSet,
			Time:     inItem.Time,
		})
	}
	return ret
}

func Convert_libvirt_DomainStatsNet_To_stats_DomainStatsNet(in []libvirt.DomainStatsNet) []stats.DomainStatsNet {
	ret := make([]stats.DomainStatsNet, 0, len(in))
	for _, inItem := range in {
		ret = append(ret, stats.DomainStatsNet{
			NameSet:    inItem.NameSet,
			Name:       inItem.Name,
			RxBytesSet: inItem.RxBytesSet,
			RxBytes:    inItem.RxBytes,
			RxPktsSet:  inItem.RxPktsSet,
			RxPkts:     inItem.RxPkts,
			RxErrsSet:  inItem.RxErrsSet,
			RxErrs:     inItem.RxErrs,
			RxDropSet:  inItem.RxDropSet,
			RxDrop:     inItem.RxDrop,
			TxBytesSet: inItem.TxBytesSet,
			TxBytes:    inItem.TxBytes,
			TxPktsSet:  inItem.TxPktsSet,
			TxPkts:     inItem.TxPkts,
			TxErrsSet:  inItem.TxErrsSet,
			TxErrs:     inItem.TxErrs,
			TxDropSet:  inItem.TxDropSet,
			TxDrop:     inItem.TxDrop,
		})
	}
	return ret
}

func Convert_libvirt_DomainStatsBlock_To_stats_DomainStatsBlock(in []libvirt.DomainStatsBlock) []stats.DomainStatsBlock {
	ret := make([]stats.DomainStatsBlock, 0, len(in))
	for _, inItem := range in {
		ret = append(ret, stats.DomainStatsBlock{
			NameSet:         inItem.NameSet,
			Name:            inItem.Name,
			BackingIndexSet: inItem.BackingIndexSet,
			BackingIndex:    inItem.BackingIndex,
			PathSet:         inItem.PathSet,
			Path:            inItem.Path,
			RdReqsSet:       inItem.RdReqsSet,
			RdReqs:          inItem.RdReqs,
			RdBytesSet:      inItem.RdBytesSet,
			RdBytes:         inItem.RdBytes,
			RdTimesSet:      inItem.RdTimesSet,
			RdTimes:         inItem.RdTimes,
			WrReqsSet:       inItem.WrReqsSet,
			WrReqs:          inItem.WrReqs,
			WrBytesSet:      inItem.WrBytesSet,
			WrBytes:         inItem.WrBytes,
			WrTimesSet:      inItem.WrTimesSet,
			WrTimes:         inItem.WrTimes,
			FlReqsSet:       inItem.FlReqsSet,
			FlReqs:          inItem.FlReqs,
			FlTimesSet:      inItem.FlTimesSet,
			FlTimes:         inItem.FlTimes,
			ErrorsSet:       inItem.ErrorsSet,
			Errors:          inItem.Errors,
			AllocationSet:   inItem.AllocationSet,
			Allocation:      inItem.Allocation,
			CapacitySet:     inItem.CapacitySet,
			Capacity:        inItem.Capacity,
			PhysicalSet:     inItem.PhysicalSet,
			Physical:        inItem.Physical,
		})
	}
	return ret
}
