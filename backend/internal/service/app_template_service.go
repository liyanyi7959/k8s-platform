package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// AppTemplateService 应用商店模板服务，负责模板的 CRUD 与内置模板初始化。
type AppTemplateService struct {
	db *gorm.DB
}

// NewAppTemplateService 创建应用商店模板服务实例。
func NewAppTemplateService(db *gorm.DB) *AppTemplateService {
	return &AppTemplateService{db: db}
}

// CreateAppTemplate 创建应用模板。会校验名称非空且不与已存在模板重名。
// 创建成功后 t.ID 会被 GORM 回填。
func (s *AppTemplateService) CreateAppTemplate(ctx context.Context, t *model.AppTemplate) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if t == nil {
		return ErrWithMessage(ErrInvalidParams, "模板参数不能为空")
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "模板名称不能为空")
	}
	t.Name = name

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.AppTemplate
		if err := tx.Where("deleted_at IS NULL AND name = ?", name).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(t).Error
	})
}

// ListAppTemplates 分页查询应用模板列表，可选按分类过滤，按 id 倒序排列。
func (s *AppTemplateService) ListAppTemplates(ctx context.Context, category string, page, pageSize int) (PageResult[model.AppTemplate], error) {
	if s.db == nil {
		return PageResult[model.AppTemplate]{}, errors.New("db is required")
	}
	page, pageSize = normalizePage(page, pageSize)

	q := s.db.WithContext(ctx).Model(&model.AppTemplate{}).Where("deleted_at IS NULL")
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[model.AppTemplate]{}, err
	}

	var rows []model.AppTemplate
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[model.AppTemplate]{}, err
	}

	return PageResult[model.AppTemplate]{List: rows, Total: int(total), Page: page, PageSize: pageSize}, nil
}

// GetAppTemplate 根据 ID 查询模板详情。
func (s *AppTemplateService) GetAppTemplate(ctx context.Context, id uint64) (*model.AppTemplate, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if id == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "模板ID无效")
	}
	var t model.AppTemplate
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// UpdateAppTemplate 根据 ID 更新模板字段。若更新名称需保证不与其他模板重名。
func (s *AppTemplateService) UpdateAppTemplate(ctx context.Context, id uint64, updates map[string]any) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模板ID无效")
	}
	if len(updates) == 0 {
		return ErrWithMessage(ErrInvalidParams, "更新内容不能为空")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AppTemplate
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// 若更新了名称，需保证名称不与其他模板冲突。
		if name, ok := updates["name"].(string); ok {
			name = strings.TrimSpace(name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "模板名称不能为空")
			}
			updates["name"] = name
			if name != row.Name {
				var existing model.AppTemplate
				if err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", name, id).First(&existing).Error; err == nil {
					return ErrConflict
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		}

		return tx.Model(&model.AppTemplate{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

// DeleteAppTemplate 软删除模板。内置模板不可删除。
func (s *AppTemplateService) DeleteAppTemplate(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模板ID无效")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AppTemplate
		if err := tx.Select("id, is_builtin").Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if row.IsBuiltin {
			return ErrWithMessage(ErrInvalidParams, "内置模板不可删除")
		}
		now := time.Now().UTC()
		return tx.Model(&model.AppTemplate{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now).Error
	})
}

// ─── 内置模板初始化 ───────────────────────────────────────────

// builtinAppTemplates 为内置应用模板列表。
var builtinAppTemplates = []model.AppTemplate{
	{
		Name:        "nginx-deploy",
		DisplayName: "Nginx",
		Description: "高性能 HTTP 服务器与反向代理，适合作为 Web 入口或静态资源服务。",
		Category:    "networking",
		Icon:        "🌐",
		IsBuiltin:   true,
		DeployType:  "yaml",
		Template: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  replicas: ${REPLICAS}
  selector:
    matchLabels:
      app: ${NAME}
  template:
    metadata:
      labels:
        app: ${NAME}
    spec:
      containers:
        - name: nginx
          image: ${IMAGE}
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
  selector:
    app: ${NAME}
`,
		Variables: `[{"name":"NAMESPACE","default":"default"},{"name":"NAME","default":"nginx"},{"name":"REPLICAS","default":"1"},{"name":"IMAGE","default":"nginx:latest"}]`,
	},
	{
		Name:        "redis-deploy",
		DisplayName: "Redis",
		Description: "高性能内存键值存储，常用作缓存、消息队列与会话存储。",
		Category:    "database",
		Icon:        "🔴",
		IsBuiltin:   true,
		DeployType:  "yaml",
		Template: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${NAME}
  template:
    metadata:
      labels:
        app: ${NAME}
    spec:
      containers:
        - name: redis
          image: ${IMAGE}
          ports:
            - containerPort: 6379
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  type: ClusterIP
  ports:
    - port: 6379
      targetPort: 6379
      protocol: TCP
  selector:
    app: ${NAME}
`,
		Variables: `[{"name":"NAMESPACE","default":"default"},{"name":"NAME","default":"redis"},{"name":"IMAGE","default":"redis:7-alpine"}]`,
	},
	{
		Name:        "mysql-deploy",
		DisplayName: "MySQL",
		Description: "主流关系型数据库，适用于各类业务系统的数据持久化。",
		Category:    "database",
		Icon:        "🗄️",
		IsBuiltin:   true,
		DeployType:  "yaml",
		Template: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${NAME}
  template:
    metadata:
      labels:
        app: ${NAME}
    spec:
      containers:
        - name: mysql
          image: ${IMAGE}
          env:
            - name: MYSQL_ROOT_PASSWORD
              value: "${ROOT_PASSWORD}"
          ports:
            - containerPort: 3306
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  type: ClusterIP
  ports:
    - port: 3306
      targetPort: 3306
      protocol: TCP
  selector:
    app: ${NAME}
`,
		Variables: `[{"name":"NAMESPACE","default":"default"},{"name":"NAME","default":"mysql"},{"name":"IMAGE","default":"mysql:8.0"},{"name":"ROOT_PASSWORD","default":"changeme"}]`,
	},
	{
		Name:        "busybox-debug",
		DisplayName: "Busybox",
		Description: "调试工具 Pod，可用于网络诊断、文件查看与临时排障。",
		Category:    "devtool",
		Icon:        "🛠️",
		IsBuiltin:   true,
		DeployType:  "yaml",
		Template: `apiVersion: v1
kind: Pod
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ${NAME}
spec:
  restartPolicy: Never
  containers:
    - name: busybox
      image: ${IMAGE}
      command: ["sleep", "3600"]
      resources:
        requests:
          cpu: 50m
          memory: 64Mi
        limits:
          cpu: 200m
          memory: 256Mi
`,
		Variables: `[{"name":"NAMESPACE","default":"default"},{"name":"NAME","default":"busybox-debug"},{"name":"IMAGE","default":"busybox:latest"}]`,
	},
	// ---- Helm 类型模板 ----
	{
		Name:           "helm-redis",
		DisplayName:    "Redis (Helm)",
		Description:    "通过 Helm Chart 部署 Bitnami Redis，适用于生产环境的高可用缓存集群。",
		Category:       "database",
		Icon:           "🔴",
		IsBuiltin:      true,
		DeployType:     "helm",
		Template:       "bitnami/redis",
		Variables:      "[]",
		HelmRepoName:   "bitnami",
		HelmRepoURL:    "https://charts.bitnami.com/bitnami",
		HelmValuesYAML: "architecture: standalone\nauth:\n  enabled: false\n",
	},
	{
		Name:           "helm-nginx",
		DisplayName:    "Nginx (Helm)",
		Description:    "通过 Helm Chart 部署 Bitnami Nginx，提供高性能 HTTP 服务器与反向代理。",
		Category:       "networking",
		Icon:           "🌐",
		IsBuiltin:      true,
		DeployType:     "helm",
		Template:       "bitnami/nginx",
		Variables:      "[]",
		HelmRepoName:   "bitnami",
		HelmRepoURL:    "https://charts.bitnami.com/bitnami",
		HelmValuesYAML: "service:\n  type: ClusterIP\n",
	},
	{
		Name:           "helm-mysql",
		DisplayName:    "MySQL (Helm)",
		Description:    "通过 Helm Chart 部署 Bitnami MySQL，适用于关系型数据库持久化场景。",
		Category:       "database",
		Icon:           "🗄️",
		IsBuiltin:      true,
		DeployType:     "helm",
		Template:       "bitnami/mysql",
		Variables:      "[]",
		HelmRepoName:   "bitnami",
		HelmRepoURL:    "https://charts.bitnami.com/bitnami",
		HelmValuesYAML: "auth:\n  rootPassword: change-me\nprimary:\n  persistence:\n    enabled: true\n",
	},
	{
		Name:           "helm-mongodb",
		DisplayName:    "MongoDB (Helm)",
		Description:    "通过 Helm Chart 部署 Bitnami MongoDB，适用于文档型数据库与大数据场景。",
		Category:       "database",
		Icon:           "🍃",
		IsBuiltin:      true,
		DeployType:     "helm",
		Template:       "bitnami/mongodb",
		Variables:      "[]",
		HelmRepoName:   "bitnami",
		HelmRepoURL:    "https://charts.bitnami.com/bitnami",
		HelmValuesYAML: "architecture: standalone\nauth:\n  enabled: false\n",
	},
	{
		Name:           "helm-prometheus",
		DisplayName:    "Prometheus (Helm)",
		Description:    "通过 Helm Chart 部署 Prometheus，用于采集、存储和查询 Kubernetes 集群监控指标。",
		Category:       "monitoring",
		Icon:           "📈",
		IsBuiltin:      true,
		DeployType:     "helm",
		Template:       "prometheus-community/prometheus",
		Variables:      "[]",
		HelmRepoName:   "prometheus-community",
		HelmRepoURL:    "https://prometheus-community.github.io/helm-charts",
		HelmValuesYAML: "server:\n  persistentVolume:\n    enabled: false\n",
	},
}

// SeedBuiltinAppTemplates 检查并初始化内置应用模板。
// 若模板已存在（按 name 判断）则跳过，保证幂等。
func (s *AppTemplateService) SeedBuiltinAppTemplates(ctx context.Context) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, t := range builtinAppTemplates {
			var existing model.AppTemplate
			err := tx.Where("deleted_at IS NULL AND name = ?", t.Name).First(&existing).Error
			if err == nil {
				// 已存在的内置 Helm 模板补齐后续版本新增的默认来源与 values，
				// 仅填充空字段，不覆盖运维人员已在应用目录中维护的配置。
				if t.DeployType == "helm" {
					updates := map[string]any{}
					if existing.HelmRepoName == "" {
						updates["helm_repo_name"] = t.HelmRepoName
					}
					if existing.HelmRepoURL == "" {
						updates["helm_repo_url"] = t.HelmRepoURL
					}
					if existing.HelmValuesYAML == "" {
						updates["helm_values_yaml"] = t.HelmValuesYAML
					}
					if len(updates) > 0 {
						if err := tx.Model(&model.AppTemplate{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
							return err
						}
					}
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// 复制一份避免闭包捕获循环变量
			tpl := t
			if err := tx.Create(&tpl).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
