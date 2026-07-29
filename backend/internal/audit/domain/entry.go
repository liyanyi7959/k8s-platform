package domain

import "time"

type Entry struct {
	ID           uint64
	UserID       uint64
	Username     string
	Action       string
	Resource     string
	ResourceName string
	ClusterID    uint64
	Namespace    string
	Path         string
	StatusCode   int
	Detail       string
	ClientIP     string
	RequestID    string
	CreatedAt    time.Time
}
