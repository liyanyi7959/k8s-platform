package ports

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerAccessRuntime is the outbound boundary for host SSH diagnostics and
// interactive terminal connections. The HTTP adapter depends on this port so
// the concrete SSH/GORM implementation remains replaceable.
type ServerAccessRuntime interface {
	ProbeServerSSH(context.Context, uint64) (SSHProbeResult, error)
	OpenServerSSH(context.Context, uint64) (*ssh.Client, string, error)
}

// SSHProbeResult is the credential-free host-fact contract returned by an SSH
// runtime. It belongs to the port because both the HTTP adapter and concrete
// SSH adapter need it without introducing an application-to-port cycle.
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

// TerminalSession is the minimal, credential-free data needed to authorize a
// short-lived server terminal connection.
type TerminalSession struct {
	UserID    uint64
	ServerID  uint64
	CreatedAt time.Time
}

// TerminalSessionStore is the lifecycle boundary for one-time terminal
// tickets. A provisioning terminal has no reason to share Kops' Pod Exec
// session store, even though both are exposed through WebSocket streams.
type TerminalSessionStore interface {
	NewSessionID() string
	Put(string, TerminalSession)
	Get(string) (TerminalSession, bool)
	Take(string) (TerminalSession, bool)
}
