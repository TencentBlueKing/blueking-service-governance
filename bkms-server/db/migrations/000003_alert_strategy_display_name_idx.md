# bkmonitor_alert_strategy displayName 唯一索引

- bkmonitor_alert_strategy 新建索引：
  - workspaceID_1_appID_1_displayName_1: AlertStrategyStoreMongo 约束同一 workspace / app 下 displayName 唯一，防止本地策略重名。
