# Kubernetes 自动化部署需求与技术设计

## 1. 建设目标

平台通过 SSH 接管空白 Linux 主机，完成主机预检、操作系统初始化、containerd 安装、Kubernetes 安装、控制平面初始化、Worker 入群、CNI 与扩展组件安装，并把可用 kubeconfig 注册到集群管理模块。用户应当能够在页面完成资源录入、方案编排、执行确认、实时观测、失败取消和幂等重试。

## 2. 本期支持边界

| 项目 | 支持范围 |
| --- | --- |
| 拓扑 | 1 个控制平面节点，0..N 个 Worker |
| 操作系统 | Debian/Ubuntu、RHEL/CentOS/Rocky/AlmaLinux 系、Kylin Linux Advanced Server V10 |
| 接入方式 | 可从后端访问的 SSH，密码或未加密私钥认证 |
| 权限 | root，或具备免密 sudo 的私钥用户；密码用户要求 SSH 密码同时可用于 sudo |
| 执行控制器 | 默认以首个 Master 作为临时 Ansible Runner；后续可替换为专用 Linux Runner |
| 容器运行时 | containerd + systemd cgroup |
| 集群引导 | kubeadm |
| CNI | Flannel、Calico、Cilium |
| 扩展组件 | metrics-server、ingress-nginx、local-storage |
| 软件来源 | 在线仓库和上游发布地址 |

多控制平面不在本期范围内。它需要稳定的 `controlPlaneEndpoint`（VIP 或负载均衡器）、etcd/证书分发以及控制平面 join 流程；在这些能力落地前，后端必须拒绝多个 Master，避免生成多个互不相关的集群。

## 3. 核心业务流程

1. 凭据管理：加密保存 SSH 密码或私钥，不向任何查询接口返回明文。
2. 服务器管理：录入地址、端口、用户和凭据；SSH 探测采集 OS、内核、CPU、内存、磁盘。
3. 部署计划：选择 Kubernetes 版本、网段、CNI、扩展组件和节点角色。
4. 部署预检：检查本地 Playbook、拓扑、CIDR、节点 SSH、Linux 类型、提权能力和硬件下限；20 GiB 磁盘建议值可由操作人员确认放行。
5. 执行门禁：预检存在错误时拒绝创建任务；通过后以数据库事务切换计划状态并异步执行。
6. Runner 调度：后端 SSH 到首个 Master，自动安装缺失的 Ansible，将 Playbook、inventory 和临时凭据上传到权限为 `0700` 的任务目录。
7. Ansible 流水线：从 Master 发起预检、系统初始化、运行时、控制平面、Worker、CNI、扩展组件和验收。
8. 结果回收：stdout/stderr 实时写入任务存储并通过 SSE 推送；成功后通过 SSH 读取 Master kubeconfig 并注册集群。
9. 清理与重试：成功、失败或取消均删除 Runner 任务目录；不自动卸载软件、执行 `kubeadm reset` 或清除目标机故障现场。
10. 可观测性：任务步骤和日志持久化，SSE 实时推送，支持取消与幂等重试。
11. 完成语义：只有 Kubernetes 节点 Ready 且 kubeconfig 成功注册到平台才标记成功；“集群已安装但注册失败”必须标记失败并给出明确日志。

## 4. API 设计

`POST /api/v1/deploy/plans/:id/preflight`

返回计划级就绪度，不包含任何凭据：

```json
{
  "ready": false,
  "checked_at": "2026-07-20T12:00:00Z",
  "checks": [
    {
      "key": "node.12.disk",
      "category": "node",
      "status": "error",
      "message": "根盘仅 17 GiB，低于建议的 20 GiB",
      "remediation": "扩容根盘，或确认容量足够后人工忽略此项",
      "ignorable": true
    }
  ]
}
```

检查状态为 `passed`、`warning` 或 `error`。只有不存在 `error` 时 `ready=true`。执行接口会在后端再次执行同一套检查，前端状态不能绕过门禁。

`POST /api/v1/deploy/plans/:id/preflight/ignore`

```json
{ "key": "node.12.disk", "ignored": true }
```

只接受后端声明为 `ignorable` 的检查键。放行记录持久化在部署计划中，并由受权限和审计中间件保护；再次预检时该项显示为 `warning/ignored`，执行前复检和 Playbook 都会遵循同一记录。

## 5. 安全与可靠性设计

- 动态 inventory 使用 YAML 和结构化序列化，避免空格、特殊字符或换行造成 INI 注入。
- inventory 与临时私钥权限为 `0600`，Runner 工作区权限为 `0700`，任务结束后统一删除；查询接口只返回脱敏视图。
- 首次接管允许自动接受 SSH host key，但限定于部署进程，不修改系统级 SSH 配置。
- Kubernetes 包版本和集群版本分离：仓库包使用 `1.x.y`，kubeadm 配置使用 `v1.x.y`。
- 不再把空 Worker 组映射为控制端 localhost。
- 所有安装动作保持可重复执行；重试不会重复执行 `kubeadm init/join`。
- 任务取消通过 context 关闭远程 SSH session，并同步计划与任务状态。
- 失败清理只处理平台生成的 Runner 工作区。目标系统变更按 Playbook 幂等语义保留，避免破坏性回滚和证据丢失。
- Kylin V10 按 RedHat/EL8 兼容族处理 containerd 仓库，避免误拼接 CentOS 10 仓库地址。

## 6. 验收标准

- Ubuntu 22.04/24.04、Rocky Linux 9 或 Kylin V10 空白主机能完成单节点和多 Worker 部署。
- 不满足 CPU 2 核、内存 2 GiB、SSH/OS/权限条件时执行接口拒绝启动；根盘低于 20 GiB 时允许人工确认放行并留存记录。
- Pod CIDR 与 Service CIDR 重叠、重复节点、非单 Master 等配置在后端被拒绝。
- 最终所有节点为 Ready，所选扩展组件完成 rollout，平台出现已注册集群。
- 日志不包含 SSH 密码、私钥或 kubeconfig 内容。
- 中途失败后可重试，中途取消后任务和计划均进入取消状态。

## 7. 后续演进任务

1. 高可用控制平面：VIP/LB、control-plane join、etcd 健康与证书生命周期。
2. 离线部署：制品清单、镜像同步、deb/rpm 仓库与完整性校验。
3. 变更治理：Playbook/变量版本快照、审批、审计和回滚策略。
4. 扩容缩容：复用预检与 join 能力，增加节点排空、reset 和资产状态回收。
5. 故障恢复：控制平面备份、etcd 快照与部署任务断点恢复。

## 8. 平台菜单与职责边界

Kubernetes 集群交付是集群生命周期的一部分，因此用户入口应位于“集群管理 / 部署 Kubernetes 集群”，并串联主机资源池、预检、部署计划、执行日志和最终的集群注册。

Ansible Playbook、Inventory 模板、软件源、环境检查等是可复用的自动化资产，应位于“自动化中心 / 运行手册资产”。它们由集群交付、应用发布、告警自愈等业务流程复用，而不直接成为某一个业务流程的入口。

SSH 密码和私钥属于平台级敏感资产，应位于“管理后台 / 平台配置 / 凭据库”，并由 `credential:read`、`credential:write`、`credential:delete` 权限保护。升级后，非管理员角色需要由管理员显式授予相应凭据权限。

菜单调整后的路径如下：

| 业务域 | 菜单 | 访问路径 |
| --- | --- | --- |
| 集群管理 | 部署 Kubernetes 集群 | `/clusters/provision` |
| 集群管理 | 主机资源池 | `/clusters/hosts` |
| 自动化中心 | 自动化总览 / 运行手册资产 | `/automation/overview`、`/automation/assets` |
| 管理后台 | 凭据库 | `/config/credentials` |

旧的 `/deploy/*` 链接保留重定向，以兼容已保存的书签和外部跳转。
