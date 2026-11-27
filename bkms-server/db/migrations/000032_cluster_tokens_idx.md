# 集群 Token 相关索引

新增两个集合，均为租户集合（`TenantScoped: true`），因此各自带 `tenantID_1`（见 `design_notes/tenant_isolation.md` 7.1）。
工作空间 ID 全局唯一，以 `workspaceID` 开头的唯一索引已等价于租户内唯一，不再加 `tenantID` 前缀。
设计说明见 `design_notes/cluster_token.md`。

## cluster_tokens

空间登记的集群 Token。

- `idx_workspaceID_fingerprint`（unique）
  `fingerprint` 是 Token 明文的 SHA-256。约束空间内 Token 不重复，同时支撑按空间列出 Token。
- `idx_workspaceID_isDefault`（unique，partial：`isDefault: true`）
  约束空间内至多一个默认 Token。项目不使用事务，并发设置默认 Token 时由该索引保证只有一个成功。

## workspace_clusters

集群在空间内的接入记录。

- `idx_workspaceID_clusterID`（unique）
  约束空间内一个集群至多接入一次，支撑按集群解析 Token。不做跨空间唯一：同一个集群可以接入多个空间，各自使用自己的 Token。
- `idx_workspaceID_tokenID`
  支撑列出、统计某个 Token 被哪些集群使用（Token 列表的集群数、删除 Token 前检查）。
