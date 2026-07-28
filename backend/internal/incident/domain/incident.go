package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound          = errors.New("incident not found")
	ErrInvalidTransition = errors.New("invalid incident transition")
	ErrVersionConflict   = errors.New("incident version conflict")
	ErrValidation        = errors.New("incident validation failed")
)

type Status string

const (
	StatusOpen             Status = "open"
	StatusAcknowledged     Status = "acknowledged"
	StatusDiagnosing       Status = "diagnosing"
	StatusAwaitingApproval Status = "awaiting_approval"
	StatusExecuting        Status = "executing"
	StatusVerifying        Status = "verifying"
	StatusResolved         Status = "resolved"
)

type Command string

const (
	CommandAcknowledge       Command = "acknowledge"
	CommandDiagnose          Command = "diagnose"
	CommandRequestApproval   Command = "request_approval"
	CommandStartExecution    Command = "start_execution"
	CommandStartVerification Command = "start_verification"
	CommandResolve           Command = "resolve"
)

type Actor struct {
	ID   uint64
	Name string
}

type Incident struct {
	ID               uint64
	AlertName        string
	ClusterID        uint64
	Namespace        string
	ResourceKind     string
	ResourceName     string
	Severity         string
	Status           Status
	Summary          string
	StartedAt        time.Time
	AcknowledgedAt   *time.Time
	ResolvedAt       *time.Time
	VerifiedAt       *time.Time
	AIConversationID *uint64
	AIProposalID     *uint64
	AssigneeID       *uint64
	AssigneeName     string
	VerificationNote string
	Version          uint64
}

type TimelineEntry struct {
	ID         uint64
	Type       string
	Title      string
	Detail     string
	OperatorID uint64
	Operator   string
	CreatedAt  time.Time
}

func (i *Incident) Execute(command Command, expectedVersion uint64, note string, actor Actor, now time.Time) (TimelineEntry, error) {
	if expectedVersion == 0 || expectedVersion != i.Version {
		return TimelineEntry{}, ErrVersionConflict
	}
	next, ok := allowedTransitions[i.Status][command]
	if !ok {
		return TimelineEntry{}, fmt.Errorf("%w: %s cannot execute %s", ErrInvalidTransition, i.Status, command)
	}

	note = strings.TrimSpace(note)
	if command == CommandResolve && i.VerifiedAt == nil && note == "" {
		return TimelineEntry{}, fmt.Errorf("%w: resolution requires verification or a manual reason", ErrValidation)
	}

	i.Status = next
	i.Version++
	switch command {
	case CommandAcknowledge:
		i.AcknowledgedAt = timePtr(now)
		if actor.ID > 0 {
			i.AssigneeID = &actor.ID
		}
		i.AssigneeName = strings.TrimSpace(actor.Name)
	case CommandStartVerification:
		i.VerificationNote = note
	case CommandResolve:
		i.ResolvedAt = timePtr(now)
		if i.VerifiedAt == nil {
			i.VerifiedAt = timePtr(now)
		}
		if note != "" {
			i.VerificationNote = note
		}
	}

	return TimelineEntry{
		Type: commandTimelineType(command), Title: statusTitle(next), Detail: note,
		OperatorID: actor.ID, Operator: strings.TrimSpace(actor.Name), CreatedAt: now,
	}, nil
}

var allowedTransitions = map[Status]map[Command]Status{
	StatusOpen:             {CommandAcknowledge: StatusAcknowledged},
	StatusAcknowledged:     {CommandDiagnose: StatusDiagnosing},
	StatusDiagnosing:       {CommandRequestApproval: StatusAwaitingApproval},
	StatusAwaitingApproval: {CommandStartExecution: StatusExecuting},
	StatusExecuting:        {CommandStartVerification: StatusVerifying},
	StatusVerifying:        {CommandResolve: StatusResolved},
}

func commandTimelineType(command Command) string { return string(command) }

func IsKnownStatus(status Status) bool {
	for candidate := range allowedTransitions {
		if status == candidate {
			return true
		}
	}
	return status == StatusResolved
}

func statusTitle(status Status) string {
	return map[Status]string{
		StatusAcknowledged: "已认领", StatusDiagnosing: "AI 诊断中",
		StatusAwaitingApproval: "等待变更确认", StatusExecuting: "自动化执行中",
		StatusVerifying: "验证中", StatusResolved: "事件已恢复",
	}[status]
}

func timePtr(value time.Time) *time.Time { return &value }
