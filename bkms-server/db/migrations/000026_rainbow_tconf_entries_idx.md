# 应用七彩石入口索引

`rainbow_tconf_entries` 存放每个应用的七彩石入口（页面地址），前端用 iframe 嵌入该地址；
一个应用至多一条记录，没有记录表示未配置。

- `idx_appID`（unique）
  支撑按应用读取、设置地址时的幂等 upsert，以及删除应用时的清理。
  应用 ID 全局唯一，因此不需要再加 `tenantID` 前缀。
