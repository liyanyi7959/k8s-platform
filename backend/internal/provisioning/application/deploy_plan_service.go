package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// DeployPlanService owns the persistence lifecycle of a provisioning plan.
type DeployPlanService struct{ repository ports.Repository }

func NewDeployPlanService(repository ports.Repository) *DeployPlanService {
	return &DeployPlanService{repository: repository}
}

type DeployPlanNodeItem struct {
	ID        uint64 `json:"id"`
	ServerID  uint64 `json:"server_id"`
	Role      string `json:"role"`
	SortOrder int    `json:"sort_order"`
}
type DeployPlanItem struct {
	ID            uint64                                            `json:"id"`
	Name          string                                            `json:"name"`
	ClusterName   string                                            `json:"cluster_name"`
	K8sVersion    string                                            `json:"k8s_version"`
	PodCIDR       string                                            `json:"pod_cidr"`
	SvcCIDR       string                                            `json:"svc_cidr"`
	CNIType       string                                            `json:"cni_type"`
	CNIConfig     map[string]any                                    `json:"cni_config,omitempty"`
	Addons        []string                                          `json:"addons,omitempty"`
	HelmInstall   bool                                              `json:"helm_install"`
	StepOverrides map[string]provisiondomain.DeployPlanStepOverride `json:"step_overrides,omitempty"`
	Status        string                                            `json:"status"`
	TaskID        *uint64                                           `json:"task_id,omitempty"`
	ClusterID     *uint64                                           `json:"cluster_id,omitempty"`
	CreatedBy     uint64                                            `json:"created_by"`
	Nodes         []DeployPlanNodeItem                              `json:"nodes,omitempty"`
	CreatedAt     string                                            `json:"created_at"`
	UpdatedAt     string                                            `json:"updated_at"`
}
type ListDeployPlansRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
}
type DeployPlanNodeRequest struct {
	ServerID  uint64 `json:"server_id"`
	Role      string `json:"role"`
	SortOrder int    `json:"sort_order"`
}
type CreateDeployPlanRequest struct {
	Name          string                                            `json:"name"`
	ClusterName   string                                            `json:"cluster_name"`
	K8sVersion    string                                            `json:"k8s_version"`
	PodCIDR       string                                            `json:"pod_cidr"`
	SvcCIDR       string                                            `json:"svc_cidr"`
	CNIType       string                                            `json:"cni_type"`
	CNIConfig     map[string]any                                    `json:"cni_config"`
	Addons        []string                                          `json:"addons"`
	HelmInstall   bool                                              `json:"helm_install"`
	StepOverrides map[string]provisiondomain.DeployPlanStepOverride `json:"step_overrides"`
	Nodes         []DeployPlanNodeRequest                           `json:"nodes"`
}
type UpdateDeployPlanRequest CreateDeployPlanRequest

func (s *DeployPlanService) List(ctx context.Context, req ListDeployPlansRequest) (PageResult[DeployPlanItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	rows, total, err := s.repository.ListDeployPlans(ctx, ports.DeployPlanListQuery{Keyword: strings.TrimSpace(req.Keyword), Status: strings.TrimSpace(req.Status), Offset: (page - 1) * pageSize, Limit: pageSize})
	if err != nil {
		return PageResult[DeployPlanItem]{}, err
	}
	items := make([]DeployPlanItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, deployPlanToItem(row, nil))
	}
	return PageResult[DeployPlanItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *DeployPlanService) Create(ctx context.Context, req CreateDeployPlanRequest, createdBy uint64) (uint64, error) {
	plan, nodes, err := normalizedPlan(req, createdBy)
	if err != nil {
		return 0, err
	}
	err = s.repository.Transaction(ctx, func(tx ports.Repository) error {
		if err := ensureUniqueClusterName(ctx, tx, plan.ClusterName, 0); err != nil {
			return err
		}
		if err := validatePlanServers(ctx, tx, nodes); err != nil {
			return err
		}
		if err := tx.CreateDeployPlan(ctx, &plan); err != nil {
			return err
		}
		for i := range nodes {
			nodes[i].PlanID = plan.ID
		}
		return tx.ReplaceDeployPlanNodes(ctx, plan.ID, nodes)
	})
	if err != nil {
		return 0, err
	}
	return plan.ID, nil
}
func (s *DeployPlanService) Get(ctx context.Context, id uint64) (DeployPlanItem, error) {
	plan, nodes, err := s.getWithNodes(ctx, id)
	if err != nil {
		return DeployPlanItem{}, err
	}
	return deployPlanToItem(plan, nodes), nil
}
func (s *DeployPlanService) Update(ctx context.Context, id uint64, req UpdateDeployPlanRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	plan, nodes, err := normalizedPlan(CreateDeployPlanRequest(req), 0)
	if err != nil {
		return err
	}
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		existing, found, err := tx.FindDeployPlan(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if existing.Status != "draft" && existing.Status != "failed" && existing.Status != "cancelled" {
			return ErrWithMessage(ErrConflict, "当前状态不允许编辑部署计划")
		}
		if err := ensureUniqueClusterName(ctx, tx, plan.ClusterName, id); err != nil {
			return err
		}
		if err := validatePlanServers(ctx, tx, nodes); err != nil {
			return err
		}
		if err := tx.UpdateDeployPlan(ctx, id, map[string]any{"name": plan.Name, "cluster_name": plan.ClusterName, "k8s_version": plan.K8sVersion, "pod_cidr": plan.PodCIDR, "svc_cidr": plan.SvcCIDR, "cni_type": plan.CNIType, "cni_config": plan.CNIConfig, "addons": plan.Addons, "helm_install": plan.HelmInstall, "step_overrides": plan.StepOverrides, "status": "draft", "task_id": nil, "cluster_id": nil}); err != nil {
			return err
		}
		for i := range nodes {
			nodes[i].PlanID = id
		}
		return tx.ReplaceDeployPlanNodes(ctx, id, nodes)
	})
}
func (s *DeployPlanService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		plan, found, err := tx.FindDeployPlan(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if plan.Status == "running" {
			return ErrWithMessage(ErrConflict, "运行中的部署计划不允许删除")
		}
		return tx.SoftDeleteDeployPlan(ctx, id, now)
	})
}
func (s *DeployPlanService) getWithNodes(ctx context.Context, id uint64) (provisiondomain.DeployPlan, []provisiondomain.DeployPlanNode, error) {
	if id == 0 {
		return provisiondomain.DeployPlan{}, nil, ErrInvalidParams
	}
	plan, found, err := s.repository.FindDeployPlan(ctx, id)
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	if !found {
		return provisiondomain.DeployPlan{}, nil, ErrNotFound
	}
	nodes, err := s.repository.ListDeployPlanNodes(ctx, id)
	return plan, nodes, err
}
func normalizedPlan(req CreateDeployPlanRequest, createdBy uint64) (provisiondomain.DeployPlan, []provisiondomain.DeployPlanNode, error) {
	input := DeployPlanInput{Name: req.Name, ClusterName: req.ClusterName, K8sVersion: req.K8sVersion, PodCIDR: req.PodCIDR, SvcCIDR: req.SvcCIDR, CNIType: req.CNIType, CNIConfig: req.CNIConfig, Addons: req.Addons, HelmInstall: req.HelmInstall, StepOverrides: req.StepOverrides, Nodes: make([]DeployPlanNodeInput, 0, len(req.Nodes))}
	for _, node := range req.Nodes {
		input.Nodes = append(input.Nodes, DeployPlanNodeInput{ServerID: node.ServerID, Role: node.Role, SortOrder: node.SortOrder})
	}
	return NormalizeDeployPlan(input, createdBy)
}
func validatePlanServers(ctx context.Context, repository ports.Repository, nodes []provisiondomain.DeployPlanNode) error {
	ids := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ServerID)
	}
	count, err := repository.CountAvailableServers(ctx, ids)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return ErrWithMessage(ErrInvalidParams, "存在不可用或不存在的服务器")
	}
	return nil
}
func ensureUniqueClusterName(ctx context.Context, repository ports.Repository, name string, excludeID uint64) error {
	exists, err := repository.DeployPlanClusterNameExists(ctx, name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrWithMessage(ErrConflict, "集群名称已被部署计划使用")
	}
	return nil
}
func deployPlanToItem(row provisiondomain.DeployPlan, nodes []provisiondomain.DeployPlanNode) DeployPlanItem {
	item := DeployPlanItem{ID: row.ID, Name: row.Name, ClusterName: row.ClusterName, K8sVersion: row.K8sVersion, PodCIDR: row.PodCIDR, SvcCIDR: row.SvcCIDR, CNIType: row.CNIType, CNIConfig: map[string]any(row.CNIConfig), Addons: []string(row.Addons), HelmInstall: row.HelmInstall, StepOverrides: decodePlanStepOverrides(row.StepOverrides), Status: row.Status, TaskID: row.TaskID, ClusterID: row.ClusterID, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
	if nodes != nil {
		item.Nodes = make([]DeployPlanNodeItem, 0, len(nodes))
		for _, node := range nodes {
			item.Nodes = append(item.Nodes, DeployPlanNodeItem{ID: node.ID, ServerID: node.ServerID, Role: node.Role, SortOrder: node.SortOrder})
		}
	}
	return item
}
func decodePlanStepOverrides(raw provisiondomain.JSONMap) map[string]provisiondomain.DeployPlanStepOverride {
	if len(raw) == 0 {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var result map[string]provisiondomain.DeployPlanStepOverride
	if json.Unmarshal(encoded, &result) != nil {
		return nil
	}
	return result
}
