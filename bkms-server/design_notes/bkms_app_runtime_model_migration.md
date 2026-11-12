# bkmsApp 存量迁移与旧 API 兼容

存量 trpc / taf 应用迁移为 `bkmsApp` 的步骤、旧 API 的兼容方式，以及迁移完成后待删除的兼容期代码。
设计说明见 [bkms_app_runtime_model.md](./bkms_app_runtime_model.md)。待删除清单全部完成后删除本文。

---

## 1. 目标与边界

**目标：**

- 存量 trpc / taf 应用迁移为 `bkmsApp`：先备份，再迁移，最后检查。
- 旧 API 入参、出参结构不变，内部一律按新模型处理，不再读写旧字段。
- 旧 API、旧字段加废弃标记；兼容期代码列入待删除清单，写明删除前提。

**范围外：**

| 事项 | 原因 / 去向 |
| --- | --- |
| 下线旧 API | 待 CLI 调用量统计上线、确认无人使用后再下线（另一个 PR 评审中） |
| 1.0.7 以下的旧版 CLI | 同上。旧版 CLI 不认识 `bkmsApp`，迁移会使其对存量应用不可用，正式环境迁移须在该方案落地后执行 |
| 删除兼容期代码 | 兼容版本先于迁移上线，上线时库内仍是旧数据，需要兼容期代码读取；见 §7 |
| 前端、CLI 死代码清理 | 旧字段删除后另提 PR，见 §7 |

---

## 2. 决策

| 问题 | 结论 |
| --- | --- |
| 迁移工具 | mongosh 脚本。正式环境只有一个租户，脚本不按租户过滤，不改 `tenantID` |
| 部署与迁移顺序 | 先部署兼容版本，再迁移。兼容版本同时兼容迁移前后的数据，迁移期间通过旧接口新建的应用已经是 `bkmsApp` |
| 旧创建接口传 trpc / taf | 走新的创建流程，创建 `bkmsApp` 应用；响应结构不变，`type` 返回 `bkmsApp` |
| 旧 spec 更新接口遇到 `bkmsApp` | 按新模型处理，见 §3 |
| 迁移时是否删除旧字段 | 不删除。旧 API 已不依赖旧字段，保留旧字段只为迁移失败的应用和回滚；冲突清零、确认不再回滚后执行 `cleanup.js` |
| 存量 TAF 的 language | 写 `cpp`（正式环境 1 个 TAF 应用已核对为 C++） |

---

## 3. 旧 API 兼容

| 旧接口 | 处理 |
| --- | --- |
| `POST /workspaces/:workspaceID/apps`（trpc / taf） | 入参转换后调用新的创建流程（`lifecycle`），见下表；helm / agones 不变 |
| `PUT /apps/:appID/trpc-spec`、`taf-spec` | 旧类型应用与 `bkmsApp` 应用处理相同，不读写旧字段：框架须与接口一致；启动命令、参数、脚本整体替换；`fileName` / `filePath` 非空时同步到框架 def；`trpcSpec.language` 与应用当前语言不一致时返回 400（技术栈不可变）；应用没有记录语言时不校验也不写入。旧逻辑修改语言时只写 `Workload.TrpcConfig`，会造成新旧字段不一致，不再保留 |
| `trpc-deploys`、`taf-deploys`、`trpc-build-deploys`、`taf-build-deploys` | 已复用通用处理逻辑，按 `mods.Of(app)` 核对框架，迁移后的应用可直接使用。只加废弃标记 |
| 详情 `appModelSpec.trpcSpec` / `tafSpec` | 改为由新模型计算：`language` 取 `Application.Language`，`fileName` / `filePath` 取框架 def，`fileContent` 取框架配置文件的默认内容。framework 为 trpc / taf 的 `bkmsApp` 应用也输出 |
| 列表 `type=trpc` / `taf` 筛选 | 已同时匹配旧类型与 `bkmsApp` + framework，不改 |

旧创建接口的入参转换：

| 旧入参 | 新流程参数 |
| --- | --- |
| `type` | 写 `bkmsApp`；`framework` 取 `trpc` / `taf` |
| `appModelSpec.trpcSpec.language` | `language`；TAF 固定为 `cpp` |
| `trpcSpec` / `tafSpec` 的 `fileName`、`filePath`、`fileContent` | 框架配置文件的名称、挂载目录、内容 |
| `command`、`args`、`shCommand`、`envVars`、`buildConfig` | 不变 |

---

## 4. 数据迁移

脚本位于 `scripts/bkmsapp-migration/`，用法见其 README。备份写入 `bkmsapp_migration_backups`，`migrate.js` 只迁移本批次已备份、且当前值仍与备份一致的应用。按顺序执行：

| 步骤 | 脚本 | 内容 |
| --- | --- | --- |
| 0 | — | 对 `applications`、`app_models`、`app_config_file_defs` 执行 mongodump |
| 1 备份 | `backup.js` | 把待迁移应用的原值写入 `bkmsapp_migration_backups`，记录批次号 |
| 2 迁移 | `migrate.js` | 默认 dry-run，输出待迁移数与冲突清单；`APPLY=true` 时写库 |
| 3 检查 | `verify.js` | 核对迁移结果，见下文 |
| 回滚 | `restore.js` | 按批次号恢复；迁移后被修改过的记录跳过并报告 |
| 清理 | `cleanup.js` | 删除旧字段。冲突清零、确认不再回滚后执行，不在本次上线中执行 |

### 4.1 迁移规则

只处理 `type` 为 `trpc` / `taf` 的应用，命中任一冲突即跳过：

| 冲突 | 判定 |
| --- | --- |
| 缺少 AppModel | 应用没有对应的 AppModel |
| 语言缺失或冲突 | tRPC：`trpcSpec.language` 与 `workload.trpcConfig.language` 都为空、两者不一致，或不在 go / cpp 中 |
| 类型冲突 | `workload.type` 非空且与应用类型不一致 |
| 双框架配置 | `trpcConfig` 与 `tafConfig` 都非空 |
| 新字段已存在且不一致 | 应用已有 `framework` / `language`，与推导值不同 |
| frameworkConfig 非空 | AppModel 已有非空的 `frameworkConfig` |
| 框架 def 数量不为 1 | 需先用 `repair_trpc_taf_framework_defs` 处理 |

无冲突的应用：

1. 更新 AppModel：`frameworkConfigVersion=1`。`frameworkConfig` 为空对象，与新建应用一样不落库。
2. 更新 Application：`type=bkmsApp`，写入 `framework`、`language`。Application 最后写，作为完成标记。

两步都按原值条件更新，可重复执行、可分批执行；已是 `bkmsApp` 的应用跳过。旧字段全部保留。

### 4.2 检查项

- 剩余 trpc / taf 应用数等于冲突清单条数。
- 本批次每个应用：`type=bkmsApp`；`framework`、`language` 与备份推导值一致（能力集合因此不变）；AppModel 存在且 `frameworkConfigVersion=1`；只有一条框架 def。
- 全库 `bkmsApp` 应用无目录外的语言。

### 4.3 待填：正式环境 dry-run 结果

| 项 | 结果 |
| --- | --- |
| 执行时间 | _待填_ |
| trpc / taf 应用数 | _待填_（2026-10 扫描时为 trpc 355、taf 1） |
| 可迁移数 / 冲突数 | _待填_ |
| 冲突明细与处理结论 | _待填_ |

脚本的本地验证方法见 `scripts/bkmsapp-migration/README.md`。

---

## 5. 上线步骤

| 步骤 | 内容 | 前置条件 |
| --- | --- | --- |
| 1 | 部署兼容版本 | — |
| 2 | 测试环境：mongodump → 备份 → dry-run → 迁移 → 检查，联调前端、CLI 1.0.7 | 步骤 1 |
| 3 | 正式环境：同上；冲突清单逐个人工处理后再执行迁移 | 旧版 CLI 方案已落地 |
| 4 | 执行 `cleanup.js` | 冲突清零；观察一个发布周期，确认不再回滚 |
| 5 | 按 §7 删除兼容期代码 | 步骤 4，及各项删除前提 |

**回滚：** 步骤 4 之前，代码可回滚到兼容版本之前的版本，数据用 `restore.js` 恢复；步骤 4 之后，代码仍可回滚（旧版本能读取只有新字段的应用），数据需从 mongodump 恢复。

---

## 6. 废弃标记

| 对象 | 方式 |
| --- | --- |
| `trpc-spec`、`taf-spec`、`trpc-deploys`、`taf-deploys`、`trpc-build-deploys`、`taf-build-deploys` | Swagger `@Deprecated`，说明替代接口 |
| `POST /workspaces/:workspaceID/apps` 的 trpc / taf | Swagger 说明改用 `bkmsapps` |
| `CreateAppInput.AppModelSpec`、`AppModelSpecInput.TrpcSpec` / `TafSpec`，详情的 `trpcSpec` / `tafSpec` | 字段注释标注废弃 |
| `AppTypeTRPC` / `AppTypeTAF`、`Application.TrpcSpec`、`Workload.Type` / `TrpcConfig` / `TafConfig`、旧 DTO 与兼容入口 | Go `// Deprecated: bkmsApp 迁移兼容，……`。业务代码新增引用会被 staticcheck（SA1019）拦截；现有引用逐行 `//nolint:staticcheck` 并说明原因；测试按该前缀在 `.golangci.yaml` 中豁免 |

---

## 7. 待删除清单

删除前提：

- **P1**：`cleanup.js` 已执行，库内没有旧类型和旧字段。
- **P2**：旧 API 调用量归零，旧 API 已下线。

| 对象 | 模块 | 前提 | 检查方式 |
| --- | --- | --- | --- |
| appruntime BRIDGE-ONLY：`Inspect` / `Normalize` / `Report` 与 issue code、`ValidateCapabilityStack`、`incomplete_language`、`FrameworkFromLegacyType`、`Snapshot` 旧字段 | server | P1 | `rg -n BRIDGE-ONLY pkg` 无结果 |
| `Application.TrpcSpec`、`DisplayLanguage` / `DisplayFramework` 的回退逻辑、`RuntimeSnapshot` | server | P1 | `rg -n TrpcSpec pkg/core/app/app.go` 无结果 |
| `Workload.Type` / `TrpcConfig` / `TafConfig`、`appmodel/compat.go` | server | P1 | — |
| `appruntime.AppTypeTRPC` / `AppTypeTAF`、`IsAppModelType` 的旧值、`OfType` 与列表筛选的旧值展开、`.golangci.yaml` 中的 SA1019 豁免 | server | P1、P2 | `rg -n 'bkmsApp 迁移兼容' pkg cmd` 无结果 |
| 集群组件 YAML 的 trpc / taf、`AddonAppTypesOf` 的框架匹配 | server | P1 | — |
| `repair_trpc_taf_framework_defs`、`render/migrate` 中的 `tafConfig.fileContent` 处理 | server | P1 | — |
| 旧路由及其 handler（`core/app/handler/app_compat.go`、`deploy/handler/deploy_compat.go`、`build_auto_deploy.go` 中的 trpc / taf 接口）、`serializer/app_compat.go`、`lifecycle/spec_compat.go`、Swagger；`CreateApp` 的 trpc / taf 分支；`appcfg.SyncFrameworkFileLocation` | server | P2 | 路由表无旧路径 |
| 详情的 `trpcSpec` / `tafSpec` | server | P2 | — |
| AppSpec `devMode.workPath` / `mountPath` 持久化字段 | server、数据 | — | 渲染已不读取。字段仍作为接口与开发模式处理的内存载体（由能力参数填充），`appdefaults` 仍会写入；删除需同时改 AppSpec 结构、`appdefaults` 与库内数据 |
| `dbfactory.TrpcApplication` / `TafApplication` 改为创建 `bkmsApp` | server 测试 | P1 | — |
| 旧路由用例 | Bruno | P2 | — |
| `use-spec-field.ts` 的 trpc / taf 分支、`app-type.ts` 的 `TRPC` / `TAF`、`create.vue` 路由映射、`api/modules/bkmsserver.ts` 的旧映射、集群组件页的 `hiddenAppTypes` | UI | P1、P2 | — |
| `constant.AppTypeTrpc` / `AppTypeTaf`、`UpdateAppStartCommand` 的 trpc / taf 分支、appspec 查看 / 编辑的 `TrpcSpec` / `TafSpec` | CLI | P1、P2 | — |
| `scripts/bkmsapp-migration/` | 脚本 | P1 后保留一个发布周期 | — |
