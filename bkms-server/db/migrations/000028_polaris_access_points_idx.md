# 北极星接入点相关索引

新增三个集合，均为租户集合（`TenantScoped: true`），因此各自带 `tenantID_1`（见 `design_notes/tenant_isolation.md` 7.1）。
另为已有集合 `polaris_configs` 新增一个索引。
工作空间 ID 全局唯一，以 `workspaceID` 开头的唯一索引已等价于租户内唯一，不再加 `tenantID` 前缀。
`cluster_addon_install_tasks` 的唯一索引对应集群这一物理资源，同样不加 `tenantID` 前缀，见下文。

## polaris_access_points

空间自建的北极星接入点。

- `idx_workspaceID_name`（unique）
  约束空间内接入点名称唯一，同时支撑按空间列出接入点。

## polaris_cluster_bindings

集群与接入点的绑定。

- `idx_workspaceID_clusterID`（unique）
  约束空间内一个集群至多绑定一个接入点，支撑按集群解析接入点。不做跨空间唯一：按设计一个集群不会被多个空间复用。
- `idx_workspaceID_accessPointID`
  支撑列出、统计某个接入点绑定的集群（接入点列表的 `boundClusterCount`、删除前检查）。

## cluster_addon_install_tasks

在集群上异步安装或更新 Addon 的任务记录（`clusteraddon/installtask`），北极星接入点用它批量安装 `bcs-polaris-operator`。

- `idx_clusterID_namespace_addonName_createdAt`
  支撑查询 Addon 在集群上最近的任务，用于组件状态判定。
- `idx_clusterID_namespace_addonName_running`（unique，仅 `status` 为 `running` 的文档）
  约束同一集群命名空间内的同一 Addon 同时至多一个运行中的任务：并发创建时由数据库拒绝第二个，不依赖先查后写。
  集群是物理资源，与 Redis 安装锁 `lock:cluster-addon:{clusterID}:{namespace}:{addon}` 一样不带租户，
  按 `design_notes/tenant_isolation.md` 7.1 保持全局唯一。

## polaris_configs

- `idx_accessPointID`
  支撑删除接入点前查询仍固定使用它的北极星配置。接入点 ID 全局唯一，不需要带应用或空间前缀；
  存量配置没有该字段，不影响建索引。
