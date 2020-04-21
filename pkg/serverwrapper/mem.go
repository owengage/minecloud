package serverwrapper

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func AvailableMiB() (int, error) {
	meminfo, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(meminfo))

	for scanner.Scan() {
		line := scanner.Text()

		bits := strings.SplitN(line, ":", 2)
		if bits[0] == "MemAvailable" {
			numunit := strings.SplitN(strings.TrimSpace(bits[1]), " ", 2)
			num := numunit[0]
			unit := numunit[1]

			if unit != "kB" {
				return 0, fmt.Errorf("unexpected unit: %s", unit)
			}

			kb, err := strconv.ParseInt(num, 10, 64)
			if err != nil {
				return 0, nil
			}

			return int(kb / int64(1024)), nil
		}
	}

	return 0, nil
}
