# 空间框架配置模板索引

`workspace_framework_config_templates` 存放工作空间自定义的框架配置模板（tRPC、TAF），
平台内置模板随代码发布，不入库。工作空间 ID 全局唯一，唯一索引不需要再加 `tenantID` 前缀。

- `idx_workspaceID_framework_language_name`（unique）
  同一工作空间、同一框架和语言下模板名称不重复；前缀 `workspaceID` 同时支撑列表查询，
  以及删除空间时按 `workspaceID` 清理。
- `idx_workspaceID_framework_language_default`（unique，partial：`isDefault: true`）
  每个框架和语言至多一个默认模板，并发设置默认模板时由该索引保证只有一个成功。
- `tenantID_1`
  与其它租户隔离集合一致，支撑按租户的运维查询。
