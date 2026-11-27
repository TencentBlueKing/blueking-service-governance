# bkmsapp-migration

存量 trpc / taf 应用迁移为 bkmsApp 的 mongosh 脚本。方案与规则见
[迁移与兼容方案](../../design_notes/bkms_app_runtime_model_migration.md)。

## 前提

- 已部署兼容版本：旧 API 以新模型运行，能同时读取迁移前后的数据。
- 1.0.7 以下的旧版 CLI 已按另一方案处理（旧版 CLI 不认识 `bkmsApp`）。
- 正式环境只有一个租户，脚本不按租户过滤，不改 `tenantID`。

## 参数

通过 `--eval` 在脚本前设置：

| 变量 | 说明 |
| --- | --- |
| `DB_NAME` | 库名，默认 `bkms_server` |
| `BATCH_ID` | 批次号，`backup` / `migrate` / `restore` 必填，同一次迁移使用同一个批次号 |
| `APPLY` | 为 `true` 时写库；`migrate` / `restore` / `cleanup` 默认只打印 |
| `MAX_DETAIL` | 明细最多打印条数，默认 200 |

## 步骤

```bash
URI="mongodb://<user>:<password>@<host>:<port>/?authSource=admin"
BATCH='var BATCH_ID="20261110-1"'

# 0. 导出
mongodump --uri "$URI" --db bkms_server --collection applications --out ./dump
mongodump --uri "$URI" --db bkms_server --collection app_models --out ./dump
mongodump --uri "$URI" --db bkms_server --collection app_config_file_defs --out ./dump

# 1. 备份
mongosh "$URI" --quiet --eval "$BATCH" scripts/bkmsapp-migration/backup.js

# 2. 迁移：先只打印，确认可迁移清单与冲突清单，再写库
mongosh "$URI" --quiet --eval "$BATCH" scripts/bkmsapp-migration/migrate.js
mongosh "$URI" --quiet --eval "$BATCH; var APPLY=true" scripts/bkmsapp-migration/migrate.js

# 3. 检查，结论为 PASS
mongosh "$URI" --quiet --eval "$BATCH" scripts/bkmsapp-migration/verify.js
```

冲突清单中的应用逐个人工处理后，用新的批次号重复第 1–3 步。

## 回滚与清理

| 脚本 | 用途 | 时机 |
| --- | --- | --- |
| `restore.js` | 按批次号恢复已迁移应用的 type、framework、language、frameworkConfigVersion；迁移后技术栈被修改的应用跳过 | 迁移后发现问题时 |
| `cleanup.js` | 删除 `applications.trpcSpec` 与 `app_models` 的 `workload.type` / `trpcConfig` / `tafConfig`；库内仍有 trpc / taf 应用时不执行 | 冲突清零、观察一个发布周期后。执行后 `restore.js` 不再可用，只能依赖 mongodump |

## 本地验证

`fixtures.js` 在库名以 `_fixture` 结尾的独立库中构造样本（3 个可迁移、7 类冲突、bkmsApp 与 helm 应用各 1 个）：

```bash
F='var DB_NAME="bkms_migration_fixture"'
mongosh "$URI" --quiet --eval "$F" scripts/bkmsapp-migration/fixtures.js
mongosh "$URI" --quiet --eval "$F; var BATCH_ID=\"t1\"" scripts/bkmsapp-migration/backup.js
mongosh "$URI" --quiet --eval "$F; var BATCH_ID=\"t1\"; var APPLY=true" scripts/bkmsapp-migration/migrate.js
mongosh "$URI" --quiet --eval "$F; var BATCH_ID=\"t1\"" scripts/bkmsapp-migration/verify.js
mongosh "$URI" --quiet --eval "$F; var BATCH_ID=\"t1\"; var APPLY=true" scripts/bkmsapp-migration/restore.js
```

期望：迁移 3 个、冲突 7 个，检查结论为 PASS；回滚后 3 个应用恢复为 trpc / taf；再次迁移结果相同。
