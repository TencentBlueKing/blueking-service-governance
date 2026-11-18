# 空间配置开关索引

`workspace_cfg_switches` 存放平台管理员为每个工作空间设置的配置开关（配置管理 BSCP、七彩石），
每个空间的每个开关至多一条记录，没有记录时视为关闭。

- `idx_workspaceID_key`（unique）
  支撑按空间读取全部开关（前缀 `workspaceID`）、平台管理员设置单个开关时的幂等 upsert，
  以及删除空间时按 `workspaceID` 清理。工作空间 ID 全局唯一，因此不需要再加 `tenantID` 前缀。
