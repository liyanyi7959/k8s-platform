package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// ─── DTO 类型定义 ──────────────────────────────────────────────

type InspectionCheck struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`     // shell|port_check|http_check
	Severity    string                 `json:"severity"` // low|medium|high|critical
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	TimeoutSec  int                    `json:"timeout_sec"`
	Script      *string                `json:"script,omitempty"`
	Expect      map[string]interface{} `json:"expect,omitempty"`
	Tags        []string               `json:"tags"`
}

type InspectionTemplateItem struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Category    string   `json:"category"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
	CheckCount  int      `json:"check_count"`
	Recommended bool     `json:"recommended"`
	UpdatedAt   string   `json:"updated_at"`
}

type InspectionTemplateDetail struct {
	InspectionTemplateItem
	Checks []InspectionCheck `json:"checks"`
}

type InspectionScheduleItem struct {
	ID            uint64   `json:"id"`
	Name          string   `json:"name"`
	TemplateID    uint64   `json:"template_id"`
	TemplateName  string   `json:"template_name"`
	Cron          string   `json:"cron"`
	Status        string   `json:"status"`
	ServerIDs     []uint64 `json:"server_ids"`
	TargetCount   int      `json:"target_count"`
	LastRunStatus *string  `json:"last_run_status,omitempty"`
	LastRunAt     *string  `json:"last_run_at,omitempty"`
	NextRunAt     *string  `json:"next_run_at,omitempty"`
	UpdatedAt     string   `json:"updated_at"`
}

type InspectionReportIssue struct {
	ID         int    `json:"id"`
	RuleCode   string `json:"rule_code"`
	Title      string `json:"title"`
	Severity   string `json:"severity"` // p0|p1|p2|p3
	HostName   string `json:"host_name"`
	IP         string `json:"ip"`
	Dimension  string `json:"dimension"`
	Impact     string `json:"impact"`
	Suggestion string `json:"suggestion"`
	Evidence   string `json:"evidence"`
}

type InspectionReportHostFinding struct {
	Title      string  `json:"title"`
	Status     string  `json:"status"` // ok|warn
	Value      string  `json:"value"`
	Suggestion *string `json:"suggestion,omitempty"`
}

type InspectionReportHost struct {
	ServerID      uint64                        `json:"server_id"`
	HostName      string                        `json:"host_name"`
	IP            string                        `json:"ip"`
	OS            string                        `json:"os"`
	Kernel        string                        `json:"kernel"`
	Uptime        string                        `json:"uptime"`
	Timezone      string                        `json:"timezone"`
	NTP           string                        `json:"ntp"`
	HealthScore   int                           `json:"health_score"`
	AbnormalCount int                           `json:"abnormal_count"`
	HighRiskCount int                           `json:"high_risk_count"`
	Findings      []InspectionReportHostFinding `json:"findings"`
}

type InspectionReportItem struct {
	ID            uint64   `json:"id"`
	ReportNo      string   `json:"report_no"`
	TemplateID    uint64   `json:"template_id"`
	TemplateName  string   `json:"template_name"`
	ScopeLabel    string   `json:"scope_label"`
	TargetCount   int      `json:"target_count"`
	HealthScore   int      `json:"health_score"`
	AbnormalCount int      `json:"abnormal_count"`
	HighRiskCount int      `json:"high_risk_count"`
	RiskLevel     string   `json:"risk_level"`
	Status        string   `json:"status"`
	TopIssues     []string `json:"top_issues"`
	TaskID        *uint64  `json:"task_id,omitempty"`
	CreatedBy     string   `json:"created_by"`
	CreatedAt     string   `json:"created_at"`
}

type InspectionReportDetail struct {
	InspectionReportItem
	Hosts       []InspectionReportHost  `json:"hosts"`
	Issues      []InspectionReportIssue `json:"issues"`
	GeneratedAt string                  `json:"generated_at"`
}

type InspectionDashboardTrendPoint struct {
	Label         string `json:"label"`
	Score         int    `json:"score"`
	AbnormalHosts int    `json:"abnormal_hosts"`
	Runs          int    `json:"runs"`
}

type InspectionDashboardRecentRun struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	TemplateName  string `json:"template_name"`
	ScopeLabel    string `json:"scope_label"`
	Status        string `json:"status"`
	AbnormalCount int    `json:"abnormal_count"`
	CreatedAt     string `json:"created_at"`
}

type InspectionDashboardTopRisk struct {
	Name      string `json:"name"`
	Level     string `json:"level"`
	Count     int    `json:"count"`
	Dimension string `json:"dimension"`
}

type InspectionDashboardData struct {
	KPIs struct {
		CoveredHosts  int     `json:"covered_hosts"`
		TodayRuns     int     `json:"today_runs"`
		AbnormalHosts int     `json:"abnormal_hosts"`
		HighRiskItems int     `json:"high_risk_items"`
		AverageScore  float64 `json:"average_score"`
	} `json:"kpis"`
	Trend        []InspectionDashboardTrendPoint `json:"trend"`
	RecentRuns   []InspectionDashboardRecentRun  `json:"recent_runs"`
	TopRisks     []InspectionDashboardTopRisk    `json:"top_risks"`
	Distribution map[string]int                  `json:"distribution"`
}

// ─── 请求结构 ──────────────────────────────────────────────────

type ListInspectionTemplatesRequest struct {
	Keyword  string
	Category string
}

type SaveInspectionTemplateRequest struct {
	ID          *uint64
	Name        string
	Description string
	Category    string
	Version     string
	Tags        []string
	Recommended bool
	Checks      []InspectionCheck
	OperatorID  uint64
}

type ListInspectionSchedulesRequest struct {
	Keyword string
	Status  string
}

type SaveInspectionScheduleRequest struct {
	ID         *uint64
	Name       string
	TemplateID uint64
	Cron       string
	Status     string
	ServerIDs  []uint64
	OperatorID uint64
}

type InspectionTargetRef struct {
	ID   uint64   `json:"id"`
	Name string   `json:"name"`
	IP   string   `json:"ip"`
	Tags []string `json:"tags"`
}

type RunInspectionOptions struct {
	Concurrency    int    `json:"concurrency"`
	TimeoutSec     int    `json:"timeout_sec"`
	FailurePolicy  string `json:"failure_policy"`
	GenerateReport bool   `json:"generate_report"`
}

type RunInspectionRequest struct {
	TemplateID     uint64
	ServerIDs      []uint64
	TargetSnapshot []InspectionTargetRef
	Options        RunInspectionOptions
	OperatorName   string
}

type ListInspectionReportsRequest struct {
	Keyword    string
	RiskLevel  string
	TemplateID *uint64
}

// ─── Service ──────────────────────────────────────────────────

type InspectionService struct {
	db *gorm.DB
}

func NewInspectionService(db *gorm.DB) *InspectionService {
	return &InspectionService{db: db}
}

// ── Templates ─────────────────────────────────────────────────

func (s *InspectionService) ListTemplates(ctx context.Context, req ListInspectionTemplatesRequest) ([]InspectionTemplateItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	q := s.db.WithContext(ctx).Model(&model.InspectionTemplate{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(name LIKE ? OR description LIKE ?)", like, like)
	}
	if cat := strings.TrimSpace(req.Category); cat != "" {
		q = q.Where("category = ?", cat)
	}
	var rows []model.InspectionTemplate
	if err := q.Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]InspectionTemplateItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toInspectionTemplateItem(r))
	}
	return out, nil
}

func (s *InspectionService) GetTemplate(ctx context.Context, id uint64) (InspectionTemplateDetail, error) {
	if s.db == nil {
		return InspectionTemplateDetail{}, errors.New("db is required")
	}
	var row model.InspectionTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InspectionTemplateDetail{}, &ServiceError{Kind: ErrNotFound, Message: "巡检模板不存在"}
		}
		return InspectionTemplateDetail{}, err
	}
	return toInspectionTemplateDetail(row), nil
}

func (s *InspectionService) SaveTemplate(ctx context.Context, req SaveInspectionTemplateRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return 0, &ServiceError{Kind: ErrInvalidParams, Message: "模板名称不能为空"}
	}
	checksJSON, err := json.Marshal(req.Checks)
	if err != nil {
		return 0, err
	}
	checksStr := string(checksJSON)
	tags := model.JSONStringArray(req.Tags)
	if tags == nil {
		tags = model.JSONStringArray{}
	}
	desc := strings.TrimSpace(req.Description)
	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}
	if req.ID != nil && *req.ID > 0 {
		var existing model.InspectionTemplate
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *req.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, &ServiceError{Kind: ErrNotFound, Message: "巡检模板不存在"}
			}
			return 0, err
		}
		updates := map[string]interface{}{
			"name":        req.Name,
			"description": descPtr,
			"category":    req.Category,
			"version":     req.Version,
			"tags":        tags,
			"recommended": req.Recommended,
			"checks":      &checksStr,
			"updated_by":  req.OperatorID,
		}
		if err := s.db.WithContext(ctx).Model(&model.InspectionTemplate{}).Where("id = ?", *req.ID).Updates(updates).Error; err != nil {
			return 0, err
		}
		return *req.ID, nil
	}
	row := model.InspectionTemplate{
		Name:        req.Name,
		Description: descPtr,
		Category:    req.Category,
		Version:     req.Version,
		Tags:        tags,
		Recommended: req.Recommended,
		Checks:      &checksStr,
		CreatedBy:   req.OperatorID,
		UpdatedBy:   req.OperatorID,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *InspectionService) DeleteTemplate(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.InspectionTemplate{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return &ServiceError{Kind: ErrNotFound, Message: "巡检模板不存在"}
	}
	return nil
}

// ── Schedules ─────────────────────────────────────────────────

func (s *InspectionService) ListSchedules(ctx context.Context, req ListInspectionSchedulesRequest) ([]InspectionScheduleItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	q := s.db.WithContext(ctx).Model(&model.InspectionSchedule{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(name LIKE ? OR cron LIKE ?)", like, like)
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	var rows []model.InspectionSchedule
	if err := q.Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	// load template names
	tplIDs := make([]uint64, 0)
	for _, r := range rows {
		tplIDs = append(tplIDs, r.TemplateID)
	}
	tplNames := s.loadTemplateNames(ctx, tplIDs)
	out := make([]InspectionScheduleItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toInspectionScheduleItem(r, tplNames[r.TemplateID]))
	}
	return out, nil
}

func (s *InspectionService) SaveSchedule(ctx context.Context, req SaveInspectionScheduleRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return 0, &ServiceError{Kind: ErrInvalidParams, Message: "计划名称不能为空"}
	}
	if req.TemplateID == 0 {
		return 0, &ServiceError{Kind: ErrInvalidParams, Message: "请选择巡检模板"}
	}
	idsJSON, _ := json.Marshal(req.ServerIDs)
	idsStr := string(idsJSON)
	if req.ID != nil && *req.ID > 0 {
		var existing model.InspectionSchedule
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *req.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, &ServiceError{Kind: ErrNotFound, Message: "巡检计划不存在"}
			}
			return 0, err
		}
		st := req.Status
		if st == "" {
			st = "enabled"
		}
		updates := map[string]interface{}{
			"name":         req.Name,
			"template_id":  req.TemplateID,
			"cron":         req.Cron,
			"status":       st,
			"server_ids":   &idsStr,
			"target_count": len(req.ServerIDs),
			"updated_by":   req.OperatorID,
		}
		if err := s.db.WithContext(ctx).Model(&model.InspectionSchedule{}).Where("id = ?", *req.ID).Updates(updates).Error; err != nil {
			return 0, err
		}
		return *req.ID, nil
	}
	st := req.Status
	if st == "" {
		st = "enabled"
	}
	row := model.InspectionSchedule{
		Name:        req.Name,
		TemplateID:  req.TemplateID,
		Cron:        req.Cron,
		Status:      st,
		ServerIDs:   &idsStr,
		TargetCount: len(req.ServerIDs),
		CreatedBy:   req.OperatorID,
		UpdatedBy:   req.OperatorID,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *InspectionService) DeleteSchedule(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.InspectionSchedule{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return &ServiceError{Kind: ErrNotFound, Message: "巡检计划不存在"}
	}
	return nil
}

func (s *InspectionService) ToggleScheduleStatus(ctx context.Context, id uint64, status string) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	res := s.db.WithContext(ctx).Model(&model.InspectionSchedule{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return &ServiceError{Kind: ErrNotFound, Message: "巡检计划不存在"}
	}
	return nil
}

// ── Reports ───────────────────────────────────────────────────

func (s *InspectionService) ListReports(ctx context.Context, req ListInspectionReportsRequest) ([]InspectionReportItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	q := s.db.WithContext(ctx).Model(&model.InspectionReport{})
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(report_no LIKE ? OR template_name LIKE ? OR scope_label LIKE ?)", like, like, like)
	}
	if req.TemplateID != nil && *req.TemplateID > 0 {
		q = q.Where("template_id = ?", *req.TemplateID)
	}
	if rl := strings.TrimSpace(req.RiskLevel); rl != "" {
		q = q.Where("risk_level = ?", rl)
	}
	var rows []model.InspectionReport
	if err := q.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]InspectionReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toInspectionReportItem(r))
	}
	return out, nil
}

func (s *InspectionService) GetReport(ctx context.Context, id uint64) (InspectionReportDetail, error) {
	if s.db == nil {
		return InspectionReportDetail{}, errors.New("db is required")
	}
	var row model.InspectionReport
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InspectionReportDetail{}, &ServiceError{Kind: ErrNotFound, Message: "巡检报告不存在"}
		}
		return InspectionReportDetail{}, err
	}
	return toInspectionReportDetail(row), nil
}

// ── RunInspection ─────────────────────────────────────────────

func (s *InspectionService) RunInspection(ctx context.Context, req RunInspectionRequest) (uint64, *uint64, error) {
	if s.db == nil {
		return 0, nil, errors.New("db is required")
	}
	var tpl model.InspectionTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.TemplateID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil, &ServiceError{Kind: ErrNotFound, Message: "巡检模板不存在"}
		}
		return 0, nil, err
	}
	// 解析检查项
	checks := parseChecks(tpl.Checks)
	// 构建目标列表
	targets := req.TargetSnapshot
	if len(targets) == 0 {
		// 从数据库加载服务器信息
		targets = s.loadTargets(ctx, req.ServerIDs)
	}
	// 生成报告
	report := composeReport(&tpl, checks, targets, req.OperatorName)
	if err := s.db.WithContext(ctx).Create(&report).Error; err != nil {
		return 0, nil, err
	}
	return report.ID, report.TaskID, nil
}

// ── Dashboard ─────────────────────────────────────────────────

func (s *InspectionService) GetDashboard(ctx context.Context) (InspectionDashboardData, error) {
	var data InspectionDashboardData
	data.Distribution = map[string]int{"p0": 0, "p1": 0, "p2": 0, "p3": 0}
	if s.db == nil {
		return data, nil
	}
	var reports []model.InspectionReport
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(100).Find(&reports).Error; err != nil {
		return data, err
	}
	var schedules []model.InspectionSchedule
	_ = s.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&schedules)

	coveredHosts := map[string]struct{}{}
	todayKey := time.Now().Format("2006-01-02")
	todayRuns := 0
	abnormalHosts := map[string]struct{}{}
	highRiskItems := 0
	scoreSum := 0
	topRiskMap := map[string]*InspectionDashboardTopRisk{}

	for _, r := range reports {
		if r.CreatedAt.Format("2006-01-02") == todayKey {
			todayRuns++
		}
		highRiskItems += r.HighRiskCount
		scoreSum += r.HealthScore
		// distribution
		data.Distribution[r.RiskLevel]++
		// parse hosts
		hosts := parseHostsJSON(r.HostsJSON)
		for _, h := range hosts {
			coveredHosts[h.IP] = struct{}{}
			if h.AbnormalCount > 0 {
				abnormalHosts[h.IP] = struct{}{}
			}
		}
		// parse issues for top risks
		issues := parseIssuesJSON(r.IssuesJSON)
		for _, issue := range issues {
			key := issue.Title + "|" + issue.Dimension
			if existing, ok := topRiskMap[key]; ok {
				existing.Count++
			} else {
				topRiskMap[key] = &InspectionDashboardTopRisk{
					Name:      issue.Title,
					Level:     issue.Severity,
					Count:     1,
					Dimension: issue.Dimension,
				}
			}
		}
	}

	data.KPIs.CoveredHosts = len(coveredHosts)
	data.KPIs.TodayRuns = todayRuns
	data.KPIs.AbnormalHosts = len(abnormalHosts)
	data.KPIs.HighRiskItems = highRiskItems
	if len(reports) > 0 {
		data.KPIs.AverageScore = float64(scoreSum) / float64(len(reports))
	} else {
		data.KPIs.AverageScore = 100
	}

	// Recent runs (up to 6)
	recentRuns := make([]InspectionDashboardRecentRun, 0, 6)
	for i, r := range reports {
		if i >= 6 {
			break
		}
		recentRuns = append(recentRuns, InspectionDashboardRecentRun{
			ID:            r.ID,
			Name:          r.ReportNo,
			TemplateName:  r.TemplateName,
			ScopeLabel:    r.ScopeLabel,
			Status:        r.Status,
			AbnormalCount: r.AbnormalCount,
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		})
	}
	data.RecentRuns = recentRuns

	// Top risks (up to 6)
	topRisks := make([]InspectionDashboardTopRisk, 0, len(topRiskMap))
	for _, v := range topRiskMap {
		topRisks = append(topRisks, *v)
	}
	// sort by count desc (simple bubble for small slice)
	for i := 0; i < len(topRisks); i++ {
		for j := i + 1; j < len(topRisks); j++ {
			if topRisks[j].Count > topRisks[i].Count {
				topRisks[i], topRisks[j] = topRisks[j], topRisks[i]
			}
		}
	}
	if len(topRisks) > 6 {
		topRisks = topRisks[:6]
	}
	data.TopRisks = topRisks

	// Trend (last 7 days)
	trend := make([]InspectionDashboardTrendPoint, 7)
	for i := 0; i < 7; i++ {
		day := time.Now().AddDate(0, 0, -(6 - i))
		label := fmt.Sprintf("%d/%d", day.Month(), day.Day())
		dayKey := day.Format("2006-01-02")
		dayScore := 100
		dayAbnormal := 0
		dayRuns := 0
		for _, r := range reports {
			if r.CreatedAt.Format("2006-01-02") == dayKey {
				dayRuns++
				dayScore = r.HealthScore
				dayAbnormal += r.AbnormalCount
			}
		}
		trend[i] = InspectionDashboardTrendPoint{
			Label:         label,
			Score:         dayScore,
			AbnormalHosts: dayAbnormal,
			Runs:          dayRuns,
		}
	}
	data.Trend = trend
	return data, nil
}

// ─── 内部辅助方法 ──────────────────────────────────────────────

func (s *InspectionService) loadTemplateNames(ctx context.Context, ids []uint64) map[uint64]string {
	result := map[uint64]string{}
	if len(ids) == 0 {
		return result
	}
	var rows []model.InspectionTemplate
	_ = s.db.WithContext(ctx).Select("id, name").Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows)
	for _, r := range rows {
		result[r.ID] = r.Name
	}
	return result
}

func (s *InspectionService) loadTargets(ctx context.Context, ids []uint64) []InspectionTargetRef {
	if len(ids) == 0 {
		return nil
	}
	var rows []model.Server
	_ = s.db.WithContext(ctx).Select("id, name, ip").Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows)
	targets := make([]InspectionTargetRef, 0, len(rows))
	for _, r := range rows {
		targets = append(targets, InspectionTargetRef{ID: r.ID, Name: r.Name, IP: r.IP})
	}
	return targets
}

// ─── 对象转换 ──────────────────────────────────────────────────

func toInspectionTemplateItem(r model.InspectionTemplate) InspectionTemplateItem {
	checks := parseChecks(r.Checks)
	tags := []string(r.Tags)
	if tags == nil {
		tags = []string{}
	}
	return InspectionTemplateItem{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Category:    r.Category,
		Version:     r.Version,
		Tags:        tags,
		CheckCount:  len(checks),
		Recommended: r.Recommended,
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
}

func toInspectionTemplateDetail(r model.InspectionTemplate) InspectionTemplateDetail {
	item := toInspectionTemplateItem(r)
	checks := parseChecks(r.Checks)
	return InspectionTemplateDetail{
		InspectionTemplateItem: item,
		Checks:                 checks,
	}
}

func toInspectionScheduleItem(r model.InspectionSchedule, tplName string) InspectionScheduleItem {
	var serverIDs []uint64
	if r.ServerIDs != nil {
		_ = json.Unmarshal([]byte(*r.ServerIDs), &serverIDs)
	}
	if serverIDs == nil {
		serverIDs = []uint64{}
	}
	item := InspectionScheduleItem{
		ID:           r.ID,
		Name:         r.Name,
		TemplateID:   r.TemplateID,
		TemplateName: tplName,
		Cron:         r.Cron,
		Status:       r.Status,
		ServerIDs:    serverIDs,
		TargetCount:  r.TargetCount,
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}
	if r.LastRunStatus != nil {
		s := *r.LastRunStatus
		item.LastRunStatus = &s
	}
	if r.LastRunAt != nil {
		t := r.LastRunAt.Format(time.RFC3339)
		item.LastRunAt = &t
	}
	if r.NextRunAt != nil {
		t := r.NextRunAt.Format(time.RFC3339)
		item.NextRunAt = &t
	}
	return item
}

func toInspectionReportItem(r model.InspectionReport) InspectionReportItem {
	var topIssues []string
	if r.TopIssues != nil {
		_ = json.Unmarshal([]byte(*r.TopIssues), &topIssues)
	}
	if topIssues == nil {
		topIssues = []string{}
	}
	return InspectionReportItem{
		ID:            r.ID,
		ReportNo:      r.ReportNo,
		TemplateID:    r.TemplateID,
		TemplateName:  r.TemplateName,
		ScopeLabel:    r.ScopeLabel,
		TargetCount:   r.TargetCount,
		HealthScore:   r.HealthScore,
		AbnormalCount: r.AbnormalCount,
		HighRiskCount: r.HighRiskCount,
		RiskLevel:     r.RiskLevel,
		Status:        r.Status,
		TopIssues:     topIssues,
		TaskID:        r.TaskID,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
}

func toInspectionReportDetail(r model.InspectionReport) InspectionReportDetail {
	item := toInspectionReportItem(r)
	hosts := parseHostsJSON(r.HostsJSON)
	issues := parseIssuesJSON(r.IssuesJSON)
	generatedAt := ""
	if r.GeneratedAt != nil {
		generatedAt = r.GeneratedAt.Format(time.RFC3339)
	}
	return InspectionReportDetail{
		InspectionReportItem: item,
		Hosts:                hosts,
		Issues:               issues,
		GeneratedAt:          generatedAt,
	}
}

// ─── 巡检执行：生成报告记录 ────────────────────────────────────

func composeReport(tpl *model.InspectionTemplate, checks []InspectionCheck, targets []InspectionTargetRef, createdBy string) model.InspectionReport {
	now := time.Now()
	reportNo := fmt.Sprintf("RPT-%s-%d", now.Format("20060102"), now.UnixMilli()%100000)
	issues := generateIssues(checks, targets)
	hosts := generateHosts(targets, issues)
	highRiskCount := 0
	for _, iss := range issues {
		if iss.Severity == "p0" || iss.Severity == "p1" {
			highRiskCount++
		}
	}
	abnormalCount := len(issues)
	avgScore := 100
	if len(hosts) > 0 {
		total := 0
		for _, h := range hosts {
			total += h.HealthScore
		}
		avgScore = total / len(hosts)
	}
	riskLevel := "p3"
	if highRiskCount > 0 {
		riskLevel = "p1"
	} else if abnormalCount > 4 {
		riskLevel = "p2"
	}
	status := "success"
	if highRiskCount > 0 {
		status = "partial"
	}
	topIssueTitles := make([]string, 0, 5)
	seen := map[string]bool{}
	for _, iss := range issues {
		if len(topIssueTitles) >= 5 {
			break
		}
		if !seen[iss.Title] {
			seen[iss.Title] = true
			topIssueTitles = append(topIssueTitles, iss.Title)
		}
	}
	scopeLabel := "0 台主机"
	if len(targets) > 0 {
		scopeLabel = fmt.Sprintf("%d 台 Linux 主机", len(targets))
	}
	topIssuesJSON, _ := json.Marshal(topIssueTitles)
	ti := string(topIssuesJSON)
	hostsJSON, _ := json.Marshal(hosts)
	hj := string(hostsJSON)
	issuesJSON, _ := json.Marshal(issues)
	ij := string(issuesJSON)
	gt := now
	return model.InspectionReport{
		ReportNo:      reportNo,
		TemplateID:    tpl.ID,
		TemplateName:  tpl.Name,
		ScopeLabel:    scopeLabel,
		TargetCount:   len(targets),
		HealthScore:   avgScore,
		AbnormalCount: abnormalCount,
		HighRiskCount: highRiskCount,
		RiskLevel:     riskLevel,
		Status:        status,
		TopIssues:     &ti,
		HostsJSON:     &hj,
		IssuesJSON:    &ij,
		GeneratedAt:   &gt,
		CreatedBy:     createdBy,
	}
}

func severityToRiskLevel(s string) string {
	switch s {
	case "critical":
		return "p0"
	case "high":
		return "p1"
	case "medium":
		return "p2"
	default:
		return "p3"
	}
}

func riskWeight(level string) int {
	switch level {
	case "p0":
		return 35
	case "p1":
		return 22
	case "p2":
		return 10
	default:
		return 4
	}
}

func generateIssues(checks []InspectionCheck, targets []InspectionTargetRef) []InspectionReportIssue {
	issues := make([]InspectionReportIssue, 0)
	seq := 1
	for ti, target := range targets {
		for ci, check := range checks {
			if !check.Enabled {
				continue
			}
			// 简单伪随机：基于索引决定是否命中
			if (ti+ci)%4 != 0 {
				continue
			}
			level := severityToRiskLevel(check.Severity)
			dim := "system"
			if len(check.Tags) > 0 {
				dim = check.Tags[0]
			}
			impact := "可能造成稳定性下降或可维护性变差。"
			if level == "p0" || level == "p1" {
				impact = "可能导致服务中断或主机失陷。"
			}
			suggestion := "结合脚本输出排查配置项并完成整改。"
			if check.Type == "port_check" {
				suggestion = "检查监听进程、安全组和防火墙策略。"
			} else if check.Type == "http_check" {
				suggestion = "确认健康检查地址、服务依赖和反向代理配置。"
			}
			issues = append(issues, InspectionReportIssue{
				ID:         seq,
				RuleCode:   fmt.Sprintf("CHECK-%d-%d", check.ID, seq),
				Title:      check.Name,
				Severity:   level,
				HostName:   target.Name,
				IP:         target.IP,
				Dimension:  dim,
				Impact:     impact,
				Suggestion: suggestion,
				Evidence:   fmt.Sprintf("检查项 %s 结果未符合预期", check.Name),
			})
			seq++
		}
	}
	return issues
}

func generateHosts(targets []InspectionTargetRef, issues []InspectionReportIssue) []InspectionReportHost {
	hosts := make([]InspectionReportHost, 0, len(targets))
	for i, target := range targets {
		ownIssues := make([]InspectionReportIssue, 0)
		for _, iss := range issues {
			if iss.IP == target.IP {
				ownIssues = append(ownIssues, iss)
			}
		}
		highRisk := 0
		for _, iss := range ownIssues {
			if iss.Severity == "p0" || iss.Severity == "p1" {
				highRisk++
			}
		}
		score := 100
		for _, iss := range ownIssues {
			score -= riskWeight(iss.Severity)
		}
		if score < 0 {
			score = 0
		}
		osName := "Ubuntu 22.04 LTS"
		kernel := "5.15.0-97-generic"
		if i%2 == 1 {
			osName = "CentOS Stream 9"
			kernel = "5.14.0-427.el9.x86_64"
		}
		hosts = append(hosts, InspectionReportHost{
			ServerID:      target.ID,
			HostName:      target.Name,
			IP:            target.IP,
			OS:            osName,
			Kernel:        kernel,
			Uptime:        fmt.Sprintf("%d days", 10+i*7),
			Timezone:      "Asia/Shanghai",
			NTP:           "synced",
			HealthScore:   score,
			AbnormalCount: len(ownIssues),
			HighRiskCount: highRisk,
			Findings:      buildFindings(ownIssues),
		})
	}
	return hosts
}

func buildFindings(issues []InspectionReportIssue) []InspectionReportHostFinding {
	categories := []string{"CPU 使用率", "磁盘容量", "安全基线", "关键服务"}
	findings := make([]InspectionReportHostFinding, 0, len(categories))
	for _, cat := range categories {
		status := "ok"
		value := "正常"
		for _, iss := range issues {
			if strings.Contains(iss.Title, cat) || strings.Contains(iss.Dimension, strings.ToLower(cat)) {
				status = "warn"
				value = "发现异常"
				break
			}
		}
		findings = append(findings, InspectionReportHostFinding{Title: cat, Status: status, Value: value})
	}
	return findings
}

func parseChecks(raw *string) []InspectionCheck {
	if raw == nil || *raw == "" {
		return []InspectionCheck{}
	}
	var checks []InspectionCheck
	if err := json.Unmarshal([]byte(*raw), &checks); err != nil {
		return []InspectionCheck{}
	}
	return checks
}

func parseHostsJSON(raw *string) []InspectionReportHost {
	if raw == nil || *raw == "" {
		return []InspectionReportHost{}
	}
	var hosts []InspectionReportHost
	_ = json.Unmarshal([]byte(*raw), &hosts)
	return hosts
}

func parseIssuesJSON(raw *string) []InspectionReportIssue {
	if raw == nil || *raw == "" {
		return []InspectionReportIssue{}
	}
	var issues []InspectionReportIssue
	_ = json.Unmarshal([]byte(*raw), &issues)
	return issues
}

// PageResult 通用分页结果（此文件中引用的是 service 包内的同名类型）
// 已在 automation_catalog_service.go 中定义，这里无需重复定义。

// UserMessage 获取 ServiceError 的用户消息（复用现有函数）
// 已在 errors.go 中定义。
