package util

import (
	"encoding/json"
	"strings"

	"libvirt.org/go/libvirt"
)

var Testdata = `[
   {
      "Vcpu" : [
         {
            "StateSet" : true,
            "TimeSet" : true,
            "State" : 1,
            "Time" : 23810000000,
            "WaitSet": false,
            "Wait": 0,
            "DelaySet": false,
            "Delay": 0
            
         },
         {
            "State" : 1,
            "StateSet" : true,
            "TimeSet" : true,
            "Time" : 17800000000,
            "WaitSet": false,
            "Wait": 0,
            "DelaySet": false,
            "Delay": 0
         },
         {
            "State" : 1,
            "StateSet" : true,
            "TimeSet" : true,
            "Time" : 23310000000,
            "WaitSet": false,
            "Wait": 0,
            "DelaySet": false,
            "Delay": 0
         },
         {
            "State" : 1,
            "StateSet" : true,
            "TimeSet" : true,
            "Time" : 17360000000,
            "WaitSet": true,
            "Wait": 1500,
            "DelaySet": true,
            "Delay": 100
         }
      ],
      "Perf" : null,
      "Cpu" : {
         "TimeSet" : true,
         "UserSet" : true,
         "SystemSet" : true,
         "System" : 27980000000,
         "User" : 1620000000,
         "Time" : 86393420788
      },
      "Domain" : {},
      "State" : null,
      "Block" : [
         {
            "ErrorsSet" : false,
            "PathSet" : true,
            "Path" : "/var/lib/libvirt/images/f28-worker-0.qcow2",
            "WrBytesSet" : true,
            "RdTimes" : 3642065293,
            "RdTimesSet" : true,
            "WrBytes" : 184719872,
            "FlTimes" : 3721268610,
            "FlTimesSet" : true,
            "BackingIndex" : 0,
            "Physical" : 41385254912,
            "Errors" : 0,
            "PhysicalSet" : true,
            "RdBytesSet" : true,
            "WrTimes" : 1374368654,
            "WrTimesSet" : true,
            "RdBytes" : 348838912,
            "Capacity" : 42949672960,
            "Allocation" : 42099802112,
            "AllocationSet" : true,
            "CapacitySet" : true,
            "WrReqsSet" : true,
            "BackingIndexSet" : false,
            "Name" : "vda",
            "NameSet" : true,
            "FlReqsSet" : true,
            "RdReqs" : 11104,
            "FlReqs" : 613,
            "RdReqsSet" : true,
            "WrReqs" : 9949
         }
      ],
      "Balloon" : null,
      "Net" : [
         {
            "RxPkts" : 15177,
            "TxPktsSet" : true,
            "RxErrs" : 0,
            "RxBytesSet" : true,
            "TxDropSet" : true,
            "RxBytes" : 29735062,
            "RxDrop" : 0,
            "TxErrsSet" : true,
            "TxDrop" : 0,
            "RxErrsSet" : true,
            "TxErrs" : 0,
            "Name" : "vnet0",
            "TxBytesSet" : true,
            "RxDropSet" : true,
            "TxBytes" : 577941,
            "RxPktsSet" : true,
            "NameSet" : true,
            "TxPkts" : 8875
         }
      ]
   }
]`

var Testdataexpected = `{
   "Block": [
     {
       "Allocation": 42099802112, 
       "AllocationSet": true, 
       "BackingIndex": 0, 
       "BackingIndexSet": false, 
       "Capacity": 42949672960, 
       "CapacitySet": true, 
       "Errors": 0, 
       "ErrorsSet": false, 
       "FlReqs": 613, 
       "FlReqsSet": true, 
       "FlTimes": 3721268610, 
       "FlTimesSet": true, 
       "Name": "vda", 
       "NameSet": true, 
       "Alias": "",
       "Path": "/var/lib/libvirt/images/f28-worker-0.qcow2", 
       "PathSet": true, 
       "Physical": 41385254912, 
       "PhysicalSet": true, 
       "RdBytes": 348838912, 
       "RdBytesSet": true, 
       "RdReqs": 11104, 
       "RdReqsSet": true, 
       "RdTimes": 3642065293, 
       "RdTimesSet": true, 
       "WrBytes": 184719872, 
       "WrBytesSet": true, 
       "WrReqs": 9949, 
       "WrReqsSet": true, 
       "WrTimes": 1374368654, 
       "WrTimesSet": true
     }
   ], 
   "Cpu": {
     "System": 27980000000, 
     "SystemSet": true, 
     "Time": 86393420788, 
     "TimeSet": true, 
     "User": 1620000000, 
     "UserSet": true
   }, 
   "Memory": {
     "ActualBalloon": 0, 
     "ActualBalloonSet": false, 
     "Available": 0, 
     "AvailableSet": false, 
     "RSS": 0, 
     "RSSSet": false, 
     "SwapIn": 0,
     "SwapInSet": false,
     "SwapOut": 0,
     "SwapOutSet": false,
     "Unused": 0, 
     "UnusedSet": false,
     "Cached": 0, 
     "CachedSet": false,
     "MajorFault": 0,
     "MajorFaultSet": false,
     "MinorFault": 0,
     "MinorFaultSet": false,
     "Usable": 0,
     "UsableSet": false,
     "Total": 0,
     "TotalSet": false
   }, 
   "MigrateDomainJobInfo": {
     "DataProcessed": 0,
     "DataProcessedSet": false,
     "DataRemaining": 0,
     "DataRemainingSet": false,
     "MemDirtyRate": 0,
     "MemDirtyRateSet": false,
     "MemoryBpsSet": false,
     "MemoryBps": 0,
     "DiskBpsSet": false,
     "DiskBps": 0
   },
   "Name": "testName", 
   "Net": [
     {
       "Name": "vnet0", 
       "NameSet": true, 
       "Alias": "",
       "AliasSet": false,
       "RxBytes": 29735062, 
       "RxBytesSet": true, 
       "RxDrop": 0, 
       "RxDropSet": true, 
       "RxErrs": 0, 
       "RxErrsSet": true, 
       "RxPkts": 15177, 
       "RxPktsSet": true, 
       "TxBytes": 577941, 
       "TxBytesSet": true, 
       "TxDrop": 0, 
       "TxDropSet": true, 
       "TxErrs": 0, 
       "TxErrsSet": true, 
       "TxPkts": 8875, 
       "TxPktsSet": true
     }
   ], 
   "UUID": "testUUID", 
   "Vcpu": [
     {
       "State": 1, 
       "StateSet": true, 
       "Time": 23810000000, 
       "TimeSet": true,
       "WaitSet": false,
       "Wait": 0,
       "DelaySet": false,
       "Delay": 0
     }, 
     {
       "State": 1, 
       "StateSet": true, 
       "Time": 17800000000, 
       "TimeSet": true,
       "WaitSet": false,
       "Wait": 0,
       "DelaySet": false,
       "Delay": 0
       
     }, 
     {
       "State": 1, 
       "StateSet": true, 
       "Time": 23310000000, 
       "TimeSet": true,
       "WaitSet": false,
       "Wait": 0,
       "DelaySet": false,
       "Delay": 0
     }, 
     {
       "State": 1, 
       "StateSet": true, 
       "Time": 17360000000, 
       "TimeSet": true,
       "WaitSet": true,
       "Wait": 1500,
       "DelaySet": true,
       "Delay": 100
     }
   ],
   "CPUMapSet": false,
   "CPUMap": null,
   "NrVirtCpu": 0
 }`

func LoadStats() ([]libvirt.DomainStats, error) {
	ret := []libvirt.DomainStats{}
	dec := json.NewDecoder(strings.NewReader(Testdata))
	err := dec.Decode(&ret)
	return ret, err
}
