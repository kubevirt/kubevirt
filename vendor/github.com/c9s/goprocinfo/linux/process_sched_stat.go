package linux

import (
	"io/ioutil"
	"strconv"
	"strings"
)

// https://www.kernel.org/doc/Documentation/scheduler/sched-stats.txt
type ProcessSchedStat struct {
	RunTime      uint64 `json:"run_time"`      // time spent on the cpu
	RunqueueTime uint64 `json:"runqueue_time"` // time spent waiting on a runqueue
	RunPeriods   uint64 `json:"run_periods"`   // # of timeslices run on this cpu
}

func ReadProcessSchedStat(path string) (*ProcessSchedStat, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	s := string(b)
	f := strings.Fields(s)

	schedStat := ProcessSchedStat{}

	var n uint64

	for i := 0; i < len(f); i++ {

		if n, err = strconv.ParseUint(f[i], 10, 64); err != nil {
			return nil, err
		}

		switch i {
		case 0:
			schedStat.RunTime = n
		case 1:
			schedStat.RunqueueTime = n
		case 2:
			schedStat.RunPeriods = n
		}

	}
	return &schedStat, nil
}
