package ports

import (
	"context"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

// Repository is the persistence boundary for provisioning use cases.  Its
// methods use domain values and explicit query parameters so application code
// has no dependency on a database driver or ORM.
//
// A Repository can start an atomic unit of work.  The repository received by
// the callback is scoped to the transaction and must not be retained.
type Repository interface {
	Transaction(ctx context.Context, fn func(Repository) error) error

	ListAppTemplates(ctx context.Context, category string, offset, limit int) ([]provisiondomain.AppTemplate, int, error)
	FindAppTemplate(ctx context.Context, id uint64) (provisiondomain.AppTemplate, bool, error)
	AppTemplateNameExists(ctx context.Context, name string, excludeID uint64) (bool, error)
	CreateAppTemplate(ctx context.Context, template *provisiondomain.AppTemplate) error
	UpdateAppTemplate(ctx context.Context, id uint64, updates map[string]any) (bool, error)
	SoftDeleteAppTemplate(ctx context.Context, id uint64, deletedAt time.Time) (bool, error)
	SeedBuiltinAppTemplates(ctx context.Context, templates []provisiondomain.AppTemplate) error

	ListCredentials(ctx context.Context, query CredentialListQuery) ([]provisiondomain.SSHCredential, int, error)
	FindCredential(ctx context.Context, id uint64) (provisiondomain.SSHCredential, bool, error)
	CreateCredential(ctx context.Context, credential *provisiondomain.SSHCredential) error
	UpdateCredential(ctx context.Context, id uint64, updates map[string]any) (bool, error)
	SoftDeleteCredentials(ctx context.Context, ids []uint64, deletedAt time.Time) (int64, error)
	CountServersByCredential(ctx context.Context, credentialID uint64) (int, error)

	ListServers(ctx context.Context, query ServerListQuery) ([]provisiondomain.DeployServer, int, error)
	ServerSummary(ctx context.Context) (ServerSummary, error)
	FindServer(ctx context.Context, id uint64) (provisiondomain.DeployServer, bool, error)
	ServerExists(ctx context.Context, ip string, port int, excludeID uint64) (bool, error)
	CreateServer(ctx context.Context, server *provisiondomain.DeployServer) error
	UpdateServer(ctx context.Context, id uint64, updates map[string]any) error
	SoftDeleteServer(ctx context.Context, id uint64, deletedAt time.Time) error

	ListDeployPlans(ctx context.Context, query DeployPlanListQuery) ([]provisiondomain.DeployPlan, int, error)
	FindDeployPlan(ctx context.Context, id uint64) (provisiondomain.DeployPlan, bool, error)
	DeployPlanClusterNameExists(ctx context.Context, name string, excludeID uint64) (bool, error)
	CountAvailableServers(ctx context.Context, ids []uint64) (int, error)
	CreateDeployPlan(ctx context.Context, plan *provisiondomain.DeployPlan) error
	UpdateDeployPlan(ctx context.Context, id uint64, updates map[string]any) error
	ReplaceDeployPlanNodes(ctx context.Context, planID uint64, nodes []provisiondomain.DeployPlanNode) error
	ListDeployPlanNodes(ctx context.Context, planID uint64) ([]provisiondomain.DeployPlanNode, error)
	SoftDeleteDeployPlan(ctx context.Context, id uint64, deletedAt time.Time) error

	ListDeployConfigs(ctx context.Context, osTypes []string) ([]provisiondomain.DeployConfig, error)
	FindDeployConfig(ctx context.Context, id uint64) (provisiondomain.DeployConfig, bool, error)
	FindDeployConfigByKey(ctx context.Context, stepKey string, osTypes []string) ([]provisiondomain.DeployConfig, error)
	ListSupportedOSTypes(ctx context.Context) ([]string, error)
	CreateDeployConfigVersion(ctx context.Context, version *provisiondomain.DeployConfigVersion) error
	UpdateDeployConfig(ctx context.Context, id uint64, updates map[string]any) error
	ListDeployConfigVersions(ctx context.Context, configID uint64, stepKey string) ([]provisiondomain.DeployConfigVersion, error)

	ListDeployRepositories(ctx context.Context, repoType string) ([]provisiondomain.DeployRepository, error)
	FindDeployRepository(ctx context.Context, id uint64) (provisiondomain.DeployRepository, bool, error)
	CreateDeployRepository(ctx context.Context, repository *provisiondomain.DeployRepository) error
	UpdateDeployRepository(ctx context.Context, id uint64, updates map[string]any) (bool, error)
	SoftDeleteDeployRepository(ctx context.Context, id uint64, deletedAt time.Time) (bool, error)
}

type CredentialListQuery struct {
	Keyword  string
	AuthType string
	Offset   int
	Limit    int
}

type ServerListQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

type DeployPlanListQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

type ServerSummary struct {
	Total       int64
	Available   int64
	Registered  int64
	Unavailable int64
	CPUCores    int64
	MemoryMB    int64
	DiskGB      int64
}
