// Package system collects host metrics (CPU, memory, disk, network, load) for
// the dashboard using gopsutil.
package system

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"sort"
)

// Metrics is a point-in-time snapshot of host resource usage.
type Metrics struct {
	Timestamp  int64       `json:"timestamp"`
	CPUPercent float64     `json:"cpu_percent"`
	CPUCores   int         `json:"cpu_cores"`
	Load1      float64     `json:"load1"`
	Load5      float64     `json:"load5"`
	Load15     float64     `json:"load15"`
	Memory     MemoryStat  `json:"memory"`
	Swap       MemoryStat  `json:"swap"`
	Disks      []DiskStat  `json:"disks"`
	Network    NetworkStat `json:"network"`
	UptimeSecs uint64      `json:"uptime_secs"`
}

// MemoryStat holds usage of a memory pool in bytes.
type MemoryStat struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskStat holds usage of a mounted filesystem.
type DiskStat struct {
	Mount       string  `json:"mount"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkStat holds cumulative bytes transferred across all interfaces.
type NetworkStat struct {
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
}

// HostInfo describes static host details for the dashboard header.
type HostInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Platform string `json:"platform"`
	Kernel   string `json:"kernel"`
	Arch     string `json:"arch"`
	GoVer    string `json:"go_version"`
}

// Collect gathers a metrics snapshot. CPU percentage is sampled over interval.
func Collect(ctx context.Context, interval time.Duration) (Metrics, error) {
	m := Metrics{Timestamp: time.Now().Unix(), CPUCores: runtime.NumCPU()}

	if pct, err := cpu.PercentWithContext(ctx, interval, false); err == nil && len(pct) > 0 {
		m.CPUPercent = pct[0]
	}
	if l, err := load.AvgWithContext(ctx); err == nil {
		m.Load1, m.Load5, m.Load15 = l.Load1, l.Load5, l.Load15
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		m.Memory = MemoryStat{Total: vm.Total, Used: vm.Used, UsedPercent: vm.UsedPercent}
	}
	if sm, err := mem.SwapMemoryWithContext(ctx); err == nil {
		m.Swap = MemoryStat{Total: sm.Total, Used: sm.Used, UsedPercent: sm.UsedPercent}
	}
	if parts, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, p := range parts {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil || usage.Total == 0 {
				continue
			}
			m.Disks = append(m.Disks, DiskStat{
				Mount:       p.Mountpoint,
				Total:       usage.Total,
				Used:        usage.Used,
				UsedPercent: usage.UsedPercent,
			})
		}
	}
	if counters, err := net.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		m.Network = NetworkStat{BytesSent: counters[0].BytesSent, BytesRecv: counters[0].BytesRecv}
	}
	if up, err := host.UptimeWithContext(ctx); err == nil {
		m.UptimeSecs = up
	}
	return m, nil
}

// ProcessInfo summarizes a running process for the dashboard.
type ProcessInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float32 `json:"memory"`
	User   string  `json:"user"`
}

// TopProcesses returns up to limit processes sorted by memory usage descending.
func TopProcesses(ctx context.Context, limit int) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		if name == "" {
			continue
		}
		memPct, _ := p.MemoryPercentWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		username, _ := p.UsernameWithContext(ctx)
		out = append(out, ProcessInfo{
			PID:    p.Pid,
			Name:   name,
			CPU:    cpuPct,
			Memory: memPct,
			User:   username,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Memory > out[j].Memory })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Host returns static host information.
func Host(ctx context.Context) HostInfo {
	info := HostInfo{Arch: runtime.GOARCH, GoVer: runtime.Version()}
	if h, err := host.InfoWithContext(ctx); err == nil {
		info.Hostname = h.Hostname
		info.OS = h.OS
		info.Platform = h.Platform + " " + h.PlatformVersion
		info.Kernel = h.KernelVersion
	}
	return info
}
