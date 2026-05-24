# K8s 资源管理能力分析报告

## 1. 结论摘要

当前系统已经具备一套可用的 K8s 管理基础盘面，尤其是在工作负载、Pod、Job/CronJob、Service、Ingress、ConfigMap、Secret、PVC/PV 这些高频资源上，已经形成了“列表 + 搜索/筛选 + YAML + 部分详情 + 部分编辑”的主链路。

如果目标是“日常集群运维可用”，当前基础功能是够用的。

如果目标是“资源管理平台能力完整”，当前还不够，主要短板集中在四类问题：

1. 写操作权限门禁不一致，存在读权限用户仍能看到编辑/删除/执行类按钮的情况。
2. 资源创建能力覆盖面偏窄，除了 Namespace、PVC、工作负载、Service、Ingress 外，多数资源没有直达创建入口。
3. 大量资源列表仍停留在“名称 + 摘要 + AGE”的压缩视图，适合浏览，不适合运维判断。
4. 详情页、编辑器、YAML 的能力分布不统一，资源之间的操作心智不一致。

## 2. 当前系统已经具备的共性能力

主视图层已经具备以下基础设施：

- 资源树导航，按工作负载、网络、存储、配置、访问控制、扩展治理等分类组织。
- 关键字搜索、命名空间筛选、列选择、排序、分页、刷新。
- 列显示状态和表格配置持久化。
- 通用 YAML 部署入口，可作为大量资源缺少“创建表单”时的兜底。
- Manifest Apply 记录回看与再次部署。

这些能力足以支撑“看资源、找资源、刷新资源、做基础变更”的第一层诉求。

## 3. 分组评估

### 3.1 强能力资源组

#### 工作负载（Deployment / StatefulSet / DaemonSet）

列表字段完整度：高

当前字段已经覆盖 Namespace、名称、Ready、Up-to-date、Available、AGE，并针对 StatefulSet、DaemonSet 补了当前副本、更新副本、Pod 序号、Service、策略等关键信息。

动作完整度：高

已具备：

- 详情
- 编辑
- 伸缩（非 DaemonSet）
- 重启
- 批量重启
- 更新镜像
- Deployment 暂停/恢复 Rollout
- Deployment 版本历史
- YAML
- 删除
- 创建入口

评价：这是当前系统里最成熟的一组资源，已经接近可生产使用的二线运维面板。

问题：动作非常强，但权限门禁没有同步收紧，存在“功能完整但授权边界不稳”的问题。

#### Jobs / CronJobs

列表字段完整度：高

Jobs 已覆盖关联 CronJob、Completions、Active、Status、AGE；CronJobs 已覆盖 Schedule、Suspend、Active、Last Schedule、Last Successful、Concurrency。

动作完整度：高

Jobs：详情、编辑、YAML、删除、批量清理已完成。

CronJobs：详情、立即执行、暂停/恢复调度、编辑、YAML、删除。

评价：作业体系的字段和动作都比较完整，是当前平台中第二梯队的成熟能力。

#### Pods

列表字段完整度：中上

当前字段覆盖 Namespace、Pod 名称、Phase、Ready、Restarts、AGE、Node，并且补了 Warning 事件角标。

动作完整度：高

已具备：

- 详情
- 日志流
- Shell
- YAML
- 单个删除
- 批量删除
- 批量强制删除
- 批量打开日志工作台

评价：已经足以支撑常规 Pod 运维。

不足：

- 没有“编辑”能力，这对 Pod 本身是合理的。
- 缺少默认展示的 Pod IP / Host IP / QoS / Owner 等辅助判断字段。
- 行级删除按钮未按写权限隐藏，和批量区的权限逻辑不一致。

#### Nodes

列表字段完整度：中上

默认字段包括节点名、Ready、CPU、内存、AGE、kubelet 版本；隐藏字段中还有 OS Image、PodCIDR、Pods。

动作完整度：高

已具备：

- 详情
- YAML
- 停止调度（cordon）
- 恢复调度（uncordon）
- 驱逐（drain）
- 删除

评价：节点运维动作是强的，明显超过普通“只读节点列表”。

问题：

- 危险动作集中，但没有显式 canWrite 门禁。
- 缺少污点、角色、InternalIP / ExternalIP、调度状态等默认字段，做节点排障时还不够直观。

### 3.2 主流资源，基础可用但不够完整

#### Services / Ingresses

列表字段完整度：中上

Service 已覆盖 Type、ClusterIP、Ports；Ingress 已覆盖 Class、Hosts、Address。

动作完整度：中上

都具备：详情、编辑、YAML、删除。

优点：核心入口链路齐全。

不足：

- 都没有权限门禁收口。
- 都有编辑和删除，但只有 Service / Ingress 具备专门创建入口；同类网络资源如 EndpointSlice / NetworkPolicy 没有创建入口。
- Service 缺少 Endpoints 数量、External IP、Session Affinity 等字段。
- Ingress 缺少 TLS、Backend Service 数、状态更细粒度字段。

#### ConfigMaps / Secrets

列表字段完整度：中上

ConfigMap 已覆盖 namespace、name、key 数；Secret 已覆盖 namespace、name、type、数据摘要。

动作完整度：中上

ConfigMap：详情、编辑、YAML、删除。

Secret：详情、明文查看、编辑、YAML、删除。

优点：

- Secret 具备审计确认后的 reveal 能力。
- ConfigMap / Secret 编辑器已经是按 Key 的可读模式。

不足：

- 没有专门创建入口，只能依赖 YAML 部署。
- 行级编辑 / 删除未做写权限门禁。
- 列表缺少 immutable、labels 数、最近变更感知、挂载引用数等有用字段。

#### PVC / PV / StorageClass

列表字段完整度：中上

PVC 和 PV 的核心存储字段已经具备，StorageClass 也有 Provisioner、ReclaimPolicy、Default、BindingMode、Expansion。

动作完整度：中等

PVC：详情、YAML、删除、创建。

PV：详情、YAML、删除。

StorageClass：编辑、YAML、删除。

不足：

- PVC / PV 没有编辑入口。
- PVC 删除按钮未按写权限隐藏。
- PV / PVC 默认字段还可以补充 AccessMode、绑定对象、容量利用等更直接的运维信息。

#### ServiceAccount / RBAC 资源

列表字段完整度：中等

ServiceAccount 有 secrets / imagePullSecrets 数量；Role / ClusterRole 有 rules 和 summary；Binding 类有 roleRef 和 subjects。

动作完整度：中等偏上

Role、ClusterRole、RoleBinding、ClusterRoleBinding：编辑、YAML、删除，且 RBAC 写权限门禁相对完整。

ServiceAccount：编辑、YAML、删除，但没有详情页，也没有创建入口。

评价：RBAC 主链路已经成型，但仍偏向 YAML 管理和摘要管理，缺少面向策略理解的详情能力。

### 3.3 以“浏览”为主，离“可运维”还有距离的资源组

这一组资源大多复用了通用面板，只展示 Namespace/Name/Summary/AGE 或 Name/Summary/AGE：

- ReplicaSets
- NetworkPolicies
- Endpoints
- EndpointSlices
- CSIStorageCapacities
- VolumeSnapshots
- ResourceQuotas
- LimitRanges
- Leases
- CSIDrivers
- CSINodes
- VolumeAttachments
- VolumeSnapshotClasses
- VolumeSnapshotContents
- CRDs
- APIServices
- PriorityClasses
- RuntimeClasses
- ValidatingWebhookConfigurations
- MutatingWebhookConfigurations
- ValidatingAdmissionPolicies
- ValidatingAdmissionPolicyBindings

优点：

- 这些资源至少都有列表、YAML、编辑、删除中的大部分链路。
- 摘要函数不是空壳，很多已经压出了第一层有用信息。

缺点：

- 摘要列承载的信息量过大，不利于快速横向比较。
- 缺少详情页，用户只能在列表摘要和 YAML 之间跳跃。
- 对于 ResourceQuota、NetworkPolicy、EndpointSlice、Webhook、CRD 这类对象，摘要远远不够支持快速决策。

结论：当前设计更像“平台已经接通 API，但产品化视图还没有做完”。

### 3.4 特殊读多写少资源

#### Events

列表字段完整度：高

Time、Type、Reason、Namespace、Object、Message、Count 都比较实用，支持过滤。

动作完整度：低，但这是合理的。

评价：Events 本来就不应强调 CRUD，这一块主要是观测能力，现状基本合理。

#### PodMetrics

列表字段完整度：中上

已有容器数、CPU、内存、统计周期、采样时间。

动作完整度：低，但也是合理的。

提供了指标详情和跳转 Pod 详情，符合只读观测资源的定位。

## 4. 四个关键结论

### 4.1 当前平台“可用”，但还不是“完整”

如果只看常用资源，系统已经足够支撑：

- 工作负载运维
- Pod 排障
- Service / Ingress 管理
- ConfigMap / Secret 管理
- Job / CronJob 管理
- 基础存储对象查看和处理

但如果要求“所有已接入资源都具备一致的字段质量和 CRUD 体验”，当前还明显不够。

### 4.2 最大的问题不是“没有按钮”，而是“资源之间不一致”

当前最影响体验的不是单点功能缺失，而是同类资源之间差异过大：

- 有的资源有详情，有的没有。
- 有的资源有专门编辑器，有的只能进 YAML。
- 有的 YAML 可保存，有的 YAML 只能看。
- 有的资源有创建入口，有的完全没有。
- 有的资源做了权限门禁，有的危险操作直接暴露。

这会让用户很难建立稳定心智模型。

### 4.3 列表字段的主要问题在“中长尾资源”

核心资源字段已经基本够用。

真正不够的是通用治理类、扩展类、存储扩展类资源，它们被压缩成摘要后，做横向比较、批量筛查、快速判断都比较吃力。

### 4.4 CRUD 的主要短板在 Create 和 Detail，不只是在 Update/Delete

Update / Delete 其实很多资源已经有了，只是载体不同：

- 表单编辑器
- YAML 编辑器
- 专门动作

真正的缺口更集中在：

- 没有创建入口
- 没有详情页
- 没有权限门禁一致性

## 5. 高优先级问题清单

### P0

1. 统一写权限门禁。

需要优先梳理以下资源的按钮显示逻辑：

- Workloads
- Pods
- Nodes
- Namespaces
- Services
- Ingresses
- IngressClasses
- ConfigMaps
- Secrets
- PDBs
- HPAs
- PVC / PV
- ServiceAccounts

这些资源里已经存在大量编辑、删除、执行类动作，但不少面板没有显式 canWrite / canWriteRbac 控制。

### P1

2. 补齐创建入口矩阵。

建议下一批优先补：

- ConfigMap
- Secret
- ServiceAccount
- HPA
- PDB
- NetworkPolicy
- ResourceQuota
- LimitRange

原因：这些资源是平台型运维最常见的“日常配置对象”，不能长期只依赖 YAML 部署。

3. 给通用摘要面板补结构化列。

优先顺序建议：

- NetworkPolicy：policy types、pod selector、ingress/egress rule 数
- Endpoints / EndpointSlice：service、address type、endpoint 数、ready/notReady 数
- ResourceQuota：配额项数、最高使用率、接近阈值标记
- ReplicaSet：desired/ready/available
- VolumeSnapshot：ready、source PVC、class、bound content
- Webhook / AdmissionPolicy：webhook 数、failurePolicy、validation 数、policyName

4. 补详情页覆盖。

建议优先补：

- NetworkPolicy
- HPA
- PDB
- ServiceAccount
- Role / ClusterRole
- RoleBinding / ClusterRoleBinding
- ResourceQuota
- CRD

### P2

5. 统一 YAML 心智。

建议明确成两种：

- “查看 YAML” 只读
- “编辑 YAML” 可保存

不要继续沿用同一个 “YAML” 按钮承载不同行为。

6. 优化核心资源默认字段。

建议补充：

- Pods：Pod IP、Host IP、Owner
- Nodes：Roles、Scheduling、InternalIP、Taints 数
- Services：External IP、Endpoints 数
- Ingresses：TLS、Backend 数
- ConfigMaps / Secrets：immutable、labels 数、引用数
- PVC / PV：绑定关系、访问模式、容量状态

## 6. 能力成熟度判断

按当前实现，可以给出一个务实评级：

- 核心运维资源成熟度：B+
- 平台通用治理资源成熟度：C+
- CRUD 一致性成熟度：C
- 权限门禁一致性：C-
- 作为日常运维入口的整体可用性：B
- 作为“统一 K8s 资源管理平台”的完整度：B-

## 7. 最终结论

当前系统不是“基础功能不够”，而是“基础已经搭起来，但完成度不均衡”。

它已经具备一条可工作的主路径，尤其是对工作负载、Pod、作业、流量入口、配置类资源已经比较能打。

真正拉低整体水平的，是中长尾资源仍停留在摘要列表阶段，以及写权限门禁、创建入口、详情能力没有形成统一标准。

如果下一阶段要继续建设，我建议按这个顺序推进：

1. 先统一写权限门禁。
2. 再补 ConfigMap / Secret / HPA / PDB / NetworkPolicy 等高频资源的创建入口。
3. 再提升通用摘要面板的结构化字段与详情页覆盖。
4. 最后统一 YAML 查看/编辑语义。

完成这四步后，整个平台会从“可用”明显跃迁到“专业且稳定”。
