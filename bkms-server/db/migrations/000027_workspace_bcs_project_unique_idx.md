# 工作空间 BCS 项目唯一索引

- workspaces 新建索引：
  - bkSystems.bkBcsProjectCode_1 (unique, partial): `workspace.Create` / 工作空间创建流程：同一个 BCS project code 只能被一个 workspace 绑定；仅当 `bkSystems.bkBcsProjectCode` 非空时生效，用于兜底显式绑定和自动复用路径的并发竞态。
  - 索引不带 `tenantID`：字段存的是 BCS 返回的 projectCode，BCS 保证它全局唯一（多租户下为 `{tenantID}-{tenantProjectCode}`，`tenantProjectCode` 才是租户内唯一的那个），全局唯一索引与租户内唯一效果相同（见 `design_notes/tenant_isolation.md` 7.1）。

## 升级前检查

共享模式下 `bkSystems.bkBcsProjectCode` 等于蓝盾 project code。`bkci_projects.code` 从 `000001` 起已是唯一索引，正常创建路径不应出现重复；竞态、手工改数或绕过蓝盾入库的写入仍可能留下重复值。`createIndexes` 遇到重复会失败，本条 migration 不会记为完成，服务升级起不来。

执行升级前在目标库运行：

```javascript
db.workspaces.aggregate([
  { $match: { "bkSystems.bkBcsProjectCode": { $gt: "" } } },
  { $group: { _id: "$bkSystems.bkBcsProjectCode", n: { $sum: 1 }, ids: { $push: "$id" } } },
  { $match: { n: { $gt: 1 } } }
])
```

结果为空即可升级。若有重复，先人工确认每个 code 应保留的 workspace，再处理其余文档（清空或改正 `bkSystems.bkBcsProjectCode`）后重新执行本 migration。空字符串不参与唯一约束。
