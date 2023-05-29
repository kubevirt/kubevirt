package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"

	domainstats "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv/util"
)

type fakeDomainCollector struct {
}

func (fc fakeDomainCollector) Describe(_ chan<- *prometheus.Desc) {
}

// Collect needs to report all metrics to see it in docs
func (fc fakeDomainCollector) Collect(ch chan<- prometheus.Metric) {
	ps := domainstats.NewPrometheusScraper(ch)

	libstatst, err := util.LoadStats()
	if err != nil {
		panic(err)
	}

	in := &libstatst[0]
	inMem := []libvirt.DomainMemoryStat{}
	inDomInfo := &libvirt.DomainInfo{}
	jobInfo := stats.DomainJobInfo{}
	out := stats.DomainStats{}
	fs := k6tv1.VirtualMachineInstanceFileSystemList{}
	ident := statsconv.DomainIdentifier(&fakeDomainIdentifier{})
	devAliasMap := make(map[string]string)

	if err = statsconv.Convert_libvirt_DomainStats_to_stats_DomainStats(ident, in, inMem, inDomInfo, devAliasMap, &jobInfo, &out); err != nil {
		panic(err)
	}

	out.Memory.ActualBalloonSet = true
	out.Memory.UnusedSet = true
	out.Memory.CachedSet = true
	out.Memory.AvailableSet = true
	out.Memory.RSSSet = true
	out.Memory.SwapInSet = true
	out.Memory.SwapOutSet = true
	out.Memory.UsableSet = true
	out.Memory.MinorFaultSet = true
	out.Memory.MajorFaultSet = true
	out.CPUMapSet = true
	out.Cpu.SystemSet = true
	out.Cpu.UserSet = true
	out.Cpu.TimeSet = true

	fs.Items = []k6tv1.VirtualMachineInstanceFileSystem{
		{
			DiskName:       "disk1",
			MountPoint:     "/",
			FileSystemType: "EXT4",
			TotalBytes:     1000,
			UsedBytes:      10,
		},
	}

	vmi := k6tv1.VirtualMachineInstance{
		Status: k6tv1.VirtualMachineInstanceStatus{
			Phase:    k6tv1.Running,
			NodeName: "test",
		},
	}
	ps.Report("test", &vmi, &domainstats.VirtualMachineInstanceStats{DomainStats: &out, FsStats: fs})
}

type fakeDomainIdentifier struct {
}

func (*fakeDomainIdentifier) GetName() (string, error) {
	return "test", nil
}

func (*fakeDomainIdentifier) GetUUIDString() (string, error) {
	return "uuid", nil
}

func RegisterFakeDomainCollector() {
	prometheus.MustRegister(fakeDomainCollector{})
}
