package application

import (
	"context"
	"errors"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

type AppTemplateService struct{ repository ports.Repository }

func NewAppTemplateService(repository ports.Repository) *AppTemplateService {
	return &AppTemplateService{repository: repository}
}
func (s *AppTemplateService) CreateAppTemplate(ctx context.Context, template *provisiondomain.AppTemplate) error {
	if s.repository == nil {
		return errors.New("repository is required")
	}
	if template == nil {
		return ErrWithMessage(ErrInvalidParams, "模板参数不能为空")
	}
	name := strings.TrimSpace(template.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "模板名称不能为空")
	}
	template.Name = name
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		exists, err := tx.AppTemplateNameExists(ctx, name, 0)
		if err != nil {
			return err
		}
		if exists {
			return ErrConflict
		}
		return tx.CreateAppTemplate(ctx, template)
	})
}
func (s *AppTemplateService) ListAppTemplates(ctx context.Context, category string, page, pageSize int) (PageResult[provisiondomain.AppTemplate], error) {
	if s.repository == nil {
		return PageResult[provisiondomain.AppTemplate]{}, errors.New("repository is required")
	}
	page, pageSize = normalizePage(page, pageSize)
	rows, total, err := s.repository.ListAppTemplates(ctx, category, (page-1)*pageSize, pageSize)
	if err != nil {
		return PageResult[provisiondomain.AppTemplate]{}, err
	}
	return PageResult[provisiondomain.AppTemplate]{List: rows, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *AppTemplateService) GetAppTemplate(ctx context.Context, id uint64) (*provisiondomain.AppTemplate, error) {
	if s.repository == nil {
		return nil, errors.New("repository is required")
	}
	if id == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "模板 ID 无效")
	}
	template, found, err := s.repository.FindAppTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return &template, nil
}
func (s *AppTemplateService) UpdateAppTemplate(ctx context.Context, id uint64, updates map[string]any) error {
	if s.repository == nil {
		return errors.New("repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模板 ID 无效")
	}
	if len(updates) == 0 {
		return ErrWithMessage(ErrInvalidParams, "更新内容不能为空")
	}
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		current, found, err := tx.FindAppTemplate(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if name, ok := updates["name"].(string); ok {
			name = strings.TrimSpace(name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "模板名称不能为空")
			}
			updates["name"] = name
			if name != current.Name {
				exists, err := tx.AppTemplateNameExists(ctx, name, id)
				if err != nil {
					return err
				}
				if exists {
					return ErrConflict
				}
			}
		}
		updated, err := tx.UpdateAppTemplate(ctx, id, updates)
		if err != nil {
			return err
		}
		if !updated {
			return ErrNotFound
		}
		return nil
	})
}
func (s *AppTemplateService) DeleteAppTemplate(ctx context.Context, id uint64) error {
	if s.repository == nil {
		return errors.New("repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模板 ID 无效")
	}
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		template, found, err := tx.FindAppTemplate(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if template.IsBuiltin {
			return ErrWithMessage(ErrInvalidParams, "内置模板不可删除")
		}
		deleted, err := tx.SoftDeleteAppTemplate(ctx, id, time.Now().UTC())
		if err != nil {
			return err
		}
		if !deleted {
			return ErrNotFound
		}
		return nil
	})
}

var builtinAppTemplates = []provisiondomain.AppTemplate{
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

// SeedBuiltinAppTemplates is idempotent; the adapter performs the atomic check-and-create/update sequence.
func (s *AppTemplateService) SeedBuiltinAppTemplates(ctx context.Context) error {
	if s.repository == nil {
		return errors.New("repository is required")
	}
	return s.repository.SeedBuiltinAppTemplates(ctx, builtinAppTemplates)
}
