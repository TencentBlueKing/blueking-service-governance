# 自动触发镜像构建索引

契约详见 `design_notes/build_trigger_contract.md`。

- build_trigger_policies 新建索引：
  - appID_1_name_1: Build Trigger PolicyStore：同一应用下策略名唯一；
  - appID_1: Build Trigger PolicyStore：按应用列出策略，以及做数量上限校验与冲突检测时使用；
- build_trigger_records 新建索引：
  - policyID_1_triggeredAt_-1: Build Trigger RecordStore：按策略查询触发记录时支持按触发时间倒序读取；
  - policyID_1_commitID_1: Build Trigger RecordStore：回调去重时按策略 + commit 查询是否已成功构建过；**不能加唯一约束**，同一 commit 仍可能产生多条 skipped / failed 记录；

`bkci_pipelines` 不做任何索引变更。触发专用流水线要求应用级唯一，这是通过把 appID 编码进 `type`（`build-trigger-{appID}`）达成的，现有的 `workspaceID_1_type_1` 唯一索引即可覆盖。
