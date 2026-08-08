package domain

import "time"

// Revision 记录一次提案状态变更的审计痕迹。它是只读值对象，由
// Application 在每次状态转换时追加到持久化存储。
type Revision struct {
	ProposalID uint64
	FromStatus Status
	ToStatus   Status
	ActorID    uint64
	ActorName  string
	Decision   string
	CreatedAt  time.Time
}
