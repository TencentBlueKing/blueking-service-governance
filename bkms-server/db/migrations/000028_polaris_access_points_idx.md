# 北极星接入点相关索引

新增两个集合，均为租户集合（`TenantScoped: true`），因此各自带 `tenantID_1`（见 `design_notes/tenant_isolation.md` 7.1）。
另为已有集合 `polaris_configs` 新增一个索引。
工作空间 ID 全局唯一，以 `workspaceID` 开头的唯一索引已等价于租户内唯一，不再加 `tenantID` 前缀。

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

## polaris_configs

- `idx_accessPointID`
  支撑删除接入点前查询仍固定使用它的北极星配置。接入点 ID 全局唯一，不需要带应用或空间前缀；
  存量配置没有该字段，不影响建索引。
