# 工作空间构建机绑定索引

新增租户集合 `bkci_build_node_bindings`（`TenantScoped: true`），因此带 `tenantID_1`（见 `design_notes/tenant_isolation.md` 7.1）。

- `workspaceID_1`
  支撑按工作空间查询绑定。不做唯一约束：目前一个工作空间只关联一台构建机，由代码保证；后续可能支持多台。
  工作空间 ID 全局唯一，不再加 `tenantID` 前缀。
