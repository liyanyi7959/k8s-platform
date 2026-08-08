# syntax=docker/dockerfile:1
# ============================================================
# 星枢 K8S 管理平台 - 后端镜像
#
# 构建上下文：项目根目录（必须包含 backend/ 目录）
# 构建命令（在项目根目录执行）：
#   docker build -f deploy/docker/Dockerfile -t k8s-platform-backend:v1.0.0 .
#
# 关键说明：
# - 项目为模块化单体，所有模块编译进同一个 Go 进程，仅需构建这一个镜像。
# - 镜像内必须包含 backend/ansible 目录：Provisioning（部署）功能依赖
#   provisioningPlaybookDir() 在“可执行文件同级/ansible”或“工作目录/ansible”
#   找到 playbook 并打包上传到远端 Master 执行。
# - 无需内置 Ansible / ssh 二进制：代码通过 Go 原生 SSH 库连接远端，
#   由远端 Master 动态安装 ansible 后执行 playbook。
# ============================================================

# ---------- 阶段一：编译 ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src

# 先拷贝依赖清单，充分利用 Docker 构建缓存
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 拷贝后端全部源码（含 ansible/ playbook 目录）
COPY backend/ .

# 静态编译，减小运行时体积且不依赖 glibc
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---------- 阶段二：运行时 ----------
FROM alpine:3.21

LABEL org.opencontainers.image.title="k8s-platform-backend" \
      org.opencontainers.image.description="星枢 K8S 管理平台后端服务" \
      org.opencontainers.image.version="v1.0.0"

# ca-certificates：HTTPS 访问外部（AI 网关、K8s API Server 等）
# tzdata：支持通过 TZ 环境变量设置容器时区
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -u 10001 -G app -h /app appuser

WORKDIR /app

# 后端二进制
COPY --from=builder /out/api ./api
# ansible playbook 目录（必须保留，Provisioning 部署功能依赖）
COPY --from=builder /src/ansible ./ansible

# 预留运行时数据目录（AI 上传等），并授权非 root 用户写入
# 注意：外部挂载（compose volume / k8s PVC）挂到 /app/data 时
# 需通过 compose named-volume 继承权限或 k8s fsGroup 保证可写。
RUN mkdir -p /app/data && chown -R appuser:app /app

USER appuser
EXPOSE 8080

ENTRYPOINT ["./api"]
