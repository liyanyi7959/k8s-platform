// server_stats_service.go 提供通过 SSH 免 Agent 采集服务器运行指标的能力。
//
// 设计要点：
// - 通过单次 SSH Session 执行一段组合 shell 命令采集所有指标
// - 解析 /proc/stat, /proc/meminfo, df, /proc/loadavg, /proc/net/dev 等
// - 系统信息采集通过 uname, lsb_release, nproc, lscpu 等命令
// - 所有解析失败均安全降级（返回零值），不影响其他指标
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ──────────────────────────────────────────────────────────
//  Types
// ──────────────────────────────────────────────────────────

// ServerStats 服务器实时运行指标。
type ServerStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	Swap    SwapStats    `json:"swap"`
	Disks   []DiskStats  `json:"disks"`
	Load    LoadStats    `json:"load"`
	Network NetworkStats `json:"network"`
	Uptime  string       `json:"uptime"`
}

// CPUStats CPU 使用统计。
type CPUStats struct {
	UsagePercent float64 `json:"usage_percent"` // 0-100
	Cores        int     `json:"cores"`
}

// MemoryStats 内存统计。
type MemoryStats struct {
	TotalBytes     int64   `json:"total_bytes"`
	UsedBytes      int64   `json:"used_bytes"`
	AvailableBytes int64   `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

// SwapStats 交换分区统计。
type SwapStats struct {
	TotalBytes   int64   `json:"total_bytes"`
	UsedBytes    int64   `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskStats 磁盘分区统计。
type DiskStats struct {
	Filesystem   string  `json:"filesystem"`
	MountPoint   string  `json:"mount_point"`
	TotalBytes   int64   `json:"total_bytes"`
	UsedBytes    int64   `json:"used_bytes"`
	AvailBytes   int64   `json:"avail_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// LoadStats 系统负载。
type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// NetworkStats 网络统计（所有非 lo 接口合计）。
type NetworkStats struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

// ServerSysInfo 服务器系统信息。
type ServerSysInfo struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
	CPUModel    string `json:"cpu_model"`
	CPUCores    int    `json:"cpu_cores"`
	MemoryTotal string `json:"memory_total"`
	DiskTotal   string `json:"disk_total"`
	Uptime      string `json:"uptime"`
}

// ──────────────────────────────────────────────────────────
//  Service
// ──────────────────────────────────────────────────────────

// ServerStatsService 采集服务器运行指标。
type ServerStatsService struct {
	serverSvc *ServerService
}

// NewServerStatsService 创建实例。
func NewServerStatsService(serverSvc *ServerService) *ServerStatsService {
	return &ServerStatsService{serverSvc: serverSvc}
}

// dialSSH 通用 SSH 连接辅助。
func (s *ServerStatsService) dialSSH(ctx context.Context, serverID uint64) (*ssh.Client, error) {
	info, err := s.serverSvc.GetServerSSHAuth(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		authMethod = ssh.Password(info.Password)
	case "key":
		signer, err := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "私钥格式错误")
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, ErrInvalidParams
	}

	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         12 * time.Second,
	}
	client, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		return nil, normalizeSSHErr(err)
	}
	return client, nil
}

// runCmd 执行单条命令并返回 stdout。
func (s *ServerStatsService) runCmd(client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer func() { _ = sess.Close() }()
	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

// ──────────────────────────────────────────────────────────
//  GetStats — 实时监控指标
// ──────────────────────────────────────────────────────────

// statsCommand 一次性采集全部指标的 shell 脚本。
// 每个段之间用分隔符分开，便于解析。
const statsCommand = `echo "===CPU_STAT===" && head -1 /proc/stat && sleep 0.2 && head -1 /proc/stat && echo "===NPROC===" && nproc 2>/dev/null && echo "===MEMINFO===" && cat /proc/meminfo && echo "===DF===" && df -B1 --output=source,size,used,avail,pcent,target 2>/dev/null | tail -n +2 && echo "===LOADAVG===" && cat /proc/loadavg && echo "===NET===" && cat /proc/net/dev && echo "===UPTIME===" && uptime -p 2>/dev/null || cat /proc/uptime`

// GetStats 通过 SSH 采集服务器实时指标。
func (s *ServerStatsService) GetStats(ctx context.Context, serverID uint64) (ServerStats, error) {
	client, err := s.dialSSH(ctx, serverID)
	if err != nil {
		return ServerStats{}, err
	}
	defer func() { _ = client.Close() }()

	output, _ := s.runCmd(client, statsCommand)

	stats := ServerStats{}
	sections := splitSections(output)

	// CPU
	if cpuLines, ok := sections["CPU_STAT"]; ok {
		stats.CPU = parseCPU(cpuLines)
	}
	if nproc, ok := sections["NPROC"]; ok {
		n, _ := strconv.Atoi(strings.TrimSpace(nproc))
		if n > 0 {
			stats.CPU.Cores = n
		}
	}

	// Memory & Swap
	if mem, ok := sections["MEMINFO"]; ok {
		stats.Memory, stats.Swap = parseMeminfo(mem)
	}

	// Disk
	if df, ok := sections["DF"]; ok {
		stats.Disks = parseDf(df)
	}

	// Load
	if load, ok := sections["LOADAVG"]; ok {
		stats.Load = parseLoadavg(load)
	}

	// Network
	if net, ok := sections["NET"]; ok {
		stats.Network = parseNetDev(net)
	}

	// Uptime
	if up, ok := sections["UPTIME"]; ok {
		stats.Uptime = strings.TrimSpace(up)
	}

	return stats, nil
}

// ──────────────────────────────────────────────────────────
//  GetSysInfo — 系统信息采集
// ──────────────────────────────────────────────────────────

const sysinfoCommand = `echo "===HOSTNAME===" && hostname && echo "===OS===" && (cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'"' -f2 || cat /etc/redhat-release 2>/dev/null || echo "Linux") && echo "===KERNEL===" && uname -r && echo "===ARCH===" && uname -m && echo "===CPU_MODEL===" && (grep 'model name' /proc/cpuinfo 2>/dev/null | head -1 | cut -d':' -f2 || echo "Unknown") && echo "===CPU_CORES===" && nproc 2>/dev/null && echo "===MEM_TOTAL===" && grep MemTotal /proc/meminfo | awk '{print $2}' && echo "===DISK_TOTAL===" && df -B1 --total 2>/dev/null | grep total | awk '{print $2}' && echo "===UPTIME===" && (uptime -p 2>/dev/null || cat /proc/uptime)`

// GetSysInfo 采集服务器静态系统信息。
func (s *ServerStatsService) GetSysInfo(ctx context.Context, serverID uint64) (ServerSysInfo, error) {
	client, err := s.dialSSH(ctx, serverID)
	if err != nil {
		return ServerSysInfo{}, err
	}
	defer func() { _ = client.Close() }()

	output, _ := s.runCmd(client, sysinfoCommand)
	sections := splitSections(output)

	info := ServerSysInfo{}

	if v, ok := sections["HOSTNAME"]; ok {
		info.Hostname = strings.TrimSpace(v)
	}
	if v, ok := sections["OS"]; ok {
		info.OS = strings.TrimSpace(v)
	}
	if v, ok := sections["KERNEL"]; ok {
		info.Kernel = strings.TrimSpace(v)
	}
	if v, ok := sections["ARCH"]; ok {
		info.Arch = strings.TrimSpace(v)
	}
	if v, ok := sections["CPU_MODEL"]; ok {
		info.CPUModel = strings.TrimSpace(v)
	}
	if v, ok := sections["CPU_CORES"]; ok {
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		info.CPUCores = n
	}
	if v, ok := sections["MEM_TOTAL"]; ok {
		kb, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		info.MemoryTotal = formatBytes(kb * 1024)
	}
	if v, ok := sections["DISK_TOTAL"]; ok {
		b, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		info.DiskTotal = formatBytes(b)
	}
	if v, ok := sections["UPTIME"]; ok {
		info.Uptime = strings.TrimSpace(v)
	}

	return info, nil
}

// ──────────────────────────────────────────────────────────
//  Parsers
// ──────────────────────────────────────────────────────────

// splitSections 按 ===KEY=== 分隔符拆分输出。
func splitSections(output string) map[string]string {
	result := map[string]string{}
	lines := strings.Split(output, "\n")
	var currentKey string
	var buf strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "===") && strings.HasSuffix(trimmed, "===") {
			if currentKey != "" {
				result[currentKey] = buf.String()
			}
			currentKey = strings.Trim(trimmed, "=")
			buf.Reset()
		} else if currentKey != "" {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(line)
		}
	}
	if currentKey != "" {
		result[currentKey] = buf.String()
	}
	return result
}

// parseCPU 通过两次 /proc/stat 采样计算 CPU 使用率。
func parseCPU(data string) CPUStats {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		return CPUStats{}
	}
	parse := func(line string) (idle, total int64) {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			return 0, 0
		}
		var sum int64
		for i := 1; i < len(fields); i++ {
			v, _ := strconv.ParseInt(fields[i], 10, 64)
			sum += v
			if i == 4 { // idle 是第 4 列（0-indexed 第 4 个数字字段）
				idle = v
			}
		}
		return idle, sum
	}
	idle1, total1 := parse(lines[0])
	idle2, total2 := parse(lines[len(lines)-1])
	deltaTotal := total2 - total1
	deltaIdle := idle2 - idle1
	if deltaTotal <= 0 {
		return CPUStats{}
	}
	usage := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return CPUStats{UsagePercent: round2(usage)}
}

// parseMeminfo 从 /proc/meminfo 解析内存和交换分区信息。
func parseMeminfo(data string) (MemoryStats, SwapStats) {
	fields := map[string]int64{}
	for _, line := range strings.Split(data, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)
		v, _ := strconv.ParseInt(valStr, 10, 64)
		fields[key] = v * 1024 // kB → bytes
	}

	memTotal := fields["MemTotal"]
	memAvail := fields["MemAvailable"]
	if memAvail == 0 {
		memAvail = fields["MemFree"] + fields["Buffers"] + fields["Cached"]
	}
	memUsed := memTotal - memAvail
	if memUsed < 0 {
		memUsed = 0
	}
	var memPct float64
	if memTotal > 0 {
		memPct = float64(memUsed) / float64(memTotal) * 100
	}

	swapTotal := fields["SwapTotal"]
	swapFree := fields["SwapFree"]
	swapUsed := swapTotal - swapFree
	if swapUsed < 0 {
		swapUsed = 0
	}
	var swapPct float64
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100
	}

	return MemoryStats{
			TotalBytes:     memTotal,
			UsedBytes:      memUsed,
			AvailableBytes: memAvail,
			UsagePercent:   round2(memPct),
		}, SwapStats{
			TotalBytes:   swapTotal,
			UsedBytes:    swapUsed,
			UsagePercent: round2(swapPct),
		}
}

// parseDf 解析 df 输出。
func parseDf(data string) []DiskStats {
	var disks []DiskStats
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		fs := fields[0]
		// 跳过 tmpfs, devtmpfs 等虚拟文件系统
		if strings.HasPrefix(fs, "tmpfs") || strings.HasPrefix(fs, "devtmpfs") ||
			strings.HasPrefix(fs, "none") || strings.HasPrefix(fs, "udev") ||
			strings.HasPrefix(fs, "shm") || strings.Contains(fs, "overlay") {
			continue
		}
		total, _ := strconv.ParseInt(fields[1], 10, 64)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		avail, _ := strconv.ParseInt(fields[3], 10, 64)
		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.ParseFloat(pctStr, 64)
		mount := fields[5]

		disks = append(disks, DiskStats{
			Filesystem:   fs,
			MountPoint:   mount,
			TotalBytes:   total,
			UsedBytes:    used,
			AvailBytes:   avail,
			UsagePercent: round2(pct),
		})
	}
	return disks
}

// parseLoadavg 解析 /proc/loadavg。
func parseLoadavg(data string) LoadStats {
	fields := strings.Fields(strings.TrimSpace(data))
	if len(fields) < 3 {
		return LoadStats{}
	}
	l1, _ := strconv.ParseFloat(fields[0], 64)
	l5, _ := strconv.ParseFloat(fields[1], 64)
	l15, _ := strconv.ParseFloat(fields[2], 64)
	return LoadStats{Load1: l1, Load5: l5, Load15: l15}
}

// parseNetDev 解析 /proc/net/dev（累计 RX/TX 字节数，排除 lo）。
func parseNetDev(data string) NetworkStats {
	var rxTotal, txTotal int64
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 10 {
			continue
		}
		rx, _ := strconv.ParseInt(fields[0], 10, 64)
		tx, _ := strconv.ParseInt(fields[8], 10, 64)
		rxTotal += rx
		txTotal += tx
	}
	return NetworkStats{RxBytes: rxTotal, TxBytes: txTotal}
}

// ──────────────────────────────────────────────────────────
//  Helpers
// ──────────────────────────────────────────────────────────

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
