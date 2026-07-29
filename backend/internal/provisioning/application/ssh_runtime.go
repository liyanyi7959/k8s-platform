package application

// SSHProbeResult is the host-fact payload reported by the SSH runtime. It
// deliberately contains no connection credentials or transport internals.
type SSHProbeResult struct {
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	OS        string  `json:"os,omitempty"`
	OSVersion string  `json:"os_version,omitempty"`
	Kernel    string  `json:"kernel,omitempty"`
	CPUCores  *uint   `json:"cpu_cores,omitempty"`
	MemoryMB  *uint64 `json:"memory_mb,omitempty"`
	DiskGB    *uint64 `json:"disk_gb,omitempty"`
}
