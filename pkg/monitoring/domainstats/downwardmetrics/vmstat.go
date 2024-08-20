package downwardmetrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type vmStat struct {
	pswpin  uint64
	pswpout uint64
}

// readVMStat reads specific fields from the /proc/vmstat file.
// We implement it here, because it is not implemented in "github.com/prometheus/procfs"
// library.
func readVMStat(path string) (*vmStat, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &vmStat{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 {
			return nil, fmt.Errorf("malformed line: %q", s.Text())
		}

		var resultField *uint64
		switch fields[0] {
		case "pswpin":
			resultField = &(result.pswpin)
		case "pswpout":
			resultField = &(result.pswpout)
		default:
			continue
		}

		value, err := strconv.ParseUint(fields[1], 0, 64)
		if err != nil {
			return nil, err
		}

		*resultField = value
	}

	return result, nil
}
