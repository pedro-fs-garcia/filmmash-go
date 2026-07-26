package metrics

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type hostCollector struct {
	memTotal     *prometheus.Desc
	memAvailable *prometheus.Desc
	cpuCores     *prometheus.Desc
}

func NewHostCollector() prometheus.Collector {
	mt := prometheus.NewDesc(
		"system_memory_total_bytes",
		"Total system memory in bytes (from /proc/meminfo).",
		nil, nil)
	ma := prometheus.NewDesc(
		"system_memory_available_bytes",
		"Available system memory in bytes (from /proc/meminfo).",
		nil, nil)
	cpu := prometheus.NewDesc(
		"system_cpu_cores",
		"Number of logical CPUs available to the system.",
		nil, nil)

	return &hostCollector{
		memTotal:     mt,
		memAvailable: ma,
		cpuCores:     cpu,
	}
}

func (c *hostCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memTotal
	ch <- c.memAvailable
	ch <- c.cpuCores
}

func (c *hostCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.cpuCores, prometheus.GaugeValue, float64(runtime.NumCPU()))

	total, available, ok := readMemInfo()
	if !ok {
		// /proc/meminfo is Linux-only; skip the memory gauges elsewhere.
		return
	}
	ch <- prometheus.MustNewConstMetric(c.memTotal, prometheus.GaugeValue, total)
	ch <- prometheus.MustNewConstMetric(c.memAvailable, prometheus.GaugeValue, available)
}

func readMemInfo() (total, available float64, ok bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	var gotTotal, gotAvail bool
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = v * 1024
			gotTotal = true
		case "MemAvailable:":
			available = v * 1024
			gotAvail = true
		}
		if gotTotal && gotAvail {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return 0, 0, false
	}
	return total, available, gotTotal && gotAvail
}
