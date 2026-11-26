# 自动触发镜像构建：接口契约与数据模型

跨子需求的唯一契约事实源。数据模型、Store、Serializer、索引与接口字段以本文为准。

| 接口                  | 归属                   | 状态  |
|---------------------|----------------------|-----|
| 策略 CRUD / 启停 / 冲突预检 | 触发策略后端管理与冲突检测        | 已落地 |
| 触发记录查询              | 自动触发记录存储与查询          | 空实现 |
| 构建回调                | 构建回调处理、版本号生成、凭证鉴权与限流 | 空实现 |

策略 CRUD、硬冲突预检与提交复检、首条 `Ensure` / 末条 `Cleanup`、存在策略时拦截构建配置，均已落地。

## 数据模型

### 触发策略 `build_trigger_policies`（`pkg/build/trigger.Policy`）

| 字段                                    | 类型     | 约束                                  |
|---------------------------------------|--------|-------------------------------------|
| `id`                                  | string | 全局唯一，`btp-` + 随机串                   |
| `appID`                               | string | 所属应用                                |
| `name`                                | string | 应用内唯一，1–32，汉字 / 字母 / 数字 / `-` / `_` |
| `event`                               | enum   | 本期仅 `push`                          |
| `branchMatchMode`                     | enum   | `eq` / `prefix` / `all`             |
| `branchMatchValue`                    | string | 英文逗号分隔多值；`all` 时为空                  |
| `pathFilter`                          | string | 留空 = 全匹配                            |
| `status`                              | enum   | `enabled` / `disabled`              |
| `pipelineID`                          | string | 蓝盾触发专用流水线 ID                        |
| `triggerID`                           | string | 蓝盾触发器元素标识；本期留空                      |
| `creator` / `createdAt` / `updatedAt` | —      | 审计字段                                |

镜像 tag **不落在策略上**，一律读应用 `buildConfig.tagConfig`。创建 / 更新须 `tagConfig.IsAutoGenerateEnabled()`（`semver`或
`custom`），否则 `INVALID_ARGUMENT`。

索引：唯一 `appID + name`；普通 `appID`。单应用上限 5 条（含已停用），由业务层校验。

### 触发记录 `build_trigger_records`（`pkg/build/trigger.Record`）

| 字段                                     | 类型     | 约束                             |
|----------------------------------------|--------|--------------------------------|
| `policyID` / `appID`                   | string | 归属                             |
| `triggeredAt`                          | time   | 触发时间                           |
| `event`                                | enum   | 与策略同枚举                         |
| `branch` / `commitID` / `commitAuthor` | string | 事件信息                           |
| `result`                               | enum   | `built` / `skipped` / `failed` |
| `buildID`                              | string | 仅 `built` 非空                   |
| `reason`                               | string | 仅 `skipped` / `failed` 非空      |
| `createdAt`                            | time   | 创建时间                           |

索引：普通 `policyID + triggeredAt`、`policyID + commitID`。**后者不能唯一**：去重口径是「同策略同 commit 已 `built` 则跳过」，
`skipped` / `failed` 仍可多次写入。全量长期保留，无条数上限与归档。

### 构建记录扩展（`build_records`）

| 字段                | 类型     | 说明                |
|-------------------|--------|-------------------|
| `triggerType`     | enum   | `manual` / `auto` |
| `triggerPolicyID` | string | 自动触发时的策略 ID，手动为空  |

### 触发专用流水线

流水线本身只含「工蜂 Git 触发器 + 回调脚本」，不执行构建。`bkci_pipelines` 唯一索引仍是 `workspaceID + type`，应用级唯一靠把
appID 编进 `type`：

| 流水线   | `type`                          | 唯一性    |
|-------|---------------------------------|--------|
| 共享构建  | `dockerfile` / `helm-git-build` | 工作空间   |
| 用户自定义 | 流水线 ID（`p-[a-z0-9]{32}`）        | 工作空间   |
| 触发专用  | `build-trigger-{appID}`         | **应用** |

内部 API：`NewTriggerPipelineManager(workspaceID).Ensure/Cleanup(ctx, appID)`。

- 模板：`assets/pipeline_templates/build_trigger.json`（trigger `elements: []`；Git 触发器由后续同步填充）
- 回调 URL：`{httpServer.publicBaseURL}/bkms/v1/bkms-server/apps/{appID}/build-trigger-policies/callback`
- 回调凭证：蓝盾 `ACCESSTOKEN`，每应用一条；本地只存 `callbackCredentialID`，明文不回显
- `Ensure` 对已有实例 create-if-missing / 原样返回，**不做整模板 `UpdatePipeline`**（会冲掉显示名、回调脚本与已同步触发器）
- `Initialize(build-trigger-*)`：已存在则原样返回；不存在则拒绝创建，必须走 `Ensure`

## 接口

前缀 `/bkms/v1/bkms-server`，下表省略。响应外层包 `data`；策略列表不分页（`count` + `results`），触发记录分页。

| 接口   | 方法与路径                                                          | 鉴权         | 出参               |
|------|----------------------------------------------------------------|------------|------------------|
| 策略列表 | GET `/apps/{appID}/build-trigger-policies`                     | 用户票据 + 查看  | 全部策略 + 总数        |
| 创建   | POST `/apps/{appID}/build-trigger-policies`                    | 用户票据 + 构建  | 策略实体             |
| 更新   | PUT `/apps/{appID}/build-trigger-policies/{policyID}`          | 用户票据 + 构建  | 策略实体             |
| 启停   | PATCH `/apps/{appID}/build-trigger-policies/{policyID}/status` | 用户票据 + 构建  | 策略实体             |
| 删除   | DELETE `/apps/{appID}/build-trigger-policies/{policyID}`       | 用户票据 + 构建  | 204              |
| 冲突预检 | POST `/apps/{appID}/build-trigger-policies/conflict-check`     | 用户票据 + 查看  | level + 策略名 + 原因 |
| 触发记录 | GET `/apps/{appID}/build-trigger-policies/{policyID}/records`  | 用户票据 + 查看  | 记录列表 + 总数        |
| 构建回调 | POST `/apps/{appID}/build-trigger-policies/callback`           | **应用独享凭证** | 三态结果             |

除回调外均走 `auth.Required` + `perm.ValidateAppByID`。回调挂在无用户鉴权路由组。`conflict-check` / `callback` 与
`{policyID}` 可共存（gin 1.12）。

策略表单（创建 / 更新 / 预检共用）：

```json
{
  "name": "dev-auto-build",
  "event": "push",
  "branchMatchMode": "prefix",
  "branchMatchValue": "feature/,hotfix/",
  "pathFilter": "src/**"
}
```

预检额外接受 `excludeTriggerID`（**值为策略 ID**，不是蓝盾 `triggerID`）。响应：

```json
{
  "data": {
    "level": "error",
    "conflictPolicyNames": [
      "prod-build"
    ],
    "conflictReasons": [
      {
        "policyName": "prod-build",
        "overlapType": "eq_eq",
        "message": "与策略「prod-build」冲突：分支等于值相同，同一次推送会产出同名镜像并互相覆盖"
      }
    ]
  }
}
```

- `level`：`none`（两列表为空数组）/ `error`（前端禁止保存）。本期不返回 `warn`（`ConflictLevelWarn` 仅前向兼容）
- `overlapType`：`all`（任一方 `all`）/ `eq_eq`（等于值相交）/ `prefix_prefix`（前缀互相包含）/ `eq_hits_prefix`（等于值命中前缀）
- 只比相同 `event`；`pathFilter` 不参与；匹配值按逗号拆分、去空白、忽略空段、大小写敏感；已停用策略仍占冲突空间
- 预检始终 200；创建 / 更新再检，硬冲突 400 `INVALID_ARGUMENT`，文案含策略名与原因

触发记录：`result`（`built` / `skipped` / `failed`，留空不筛）+ `page` / `pageSize`。

回调请求 / 响应：

```json
{
  "policyID": "btp-xxx",
  "event": "push",
  "branch": "master",
  "commitID": "abc1234",
  "commitAuthor": "someone",
  "eventTime": "2026-08-08T15:30:00Z"
}
```

```json
{
  "data": {
    "result": "built",
    "buildID": "b-xxx",
    "reason": ""
  }
}
```

三态均 HTTP 200，靠 `result` 区分。`built` 时 `buildID` 非空、`reason` 空；`skipped` / `failed` 相反。

### 回调字段 ↔ 蓝盾变量

| 请求体字段          | 来源                                           | 说明                       |
|----------------|----------------------------------------------|--------------------------|
| `appID`        | 路径                                           | Ensure 写入 URL，不进 body    |
| `policyID`     | 触发器 `customVariables.BKMS_TRIGGER_POLICY_ID` | 每策略一个触发器；**不用** `stepId` |
| `event`        | `BKMS_TRIGGER_EVENT`，缺省 `push`               |                          |
| `branch`       | `BK_CI_REPO_GIT_WEBHOOK_BRANCH`              |                          |
| `commitID`     | `BK_CI_REPO_GIT_WEBHOOK_PUSH_HEAD_COMMIT_ID` |                          |
| `commitAuthor` | `BK_CI_REPO_GIT_WEBHOOK_PUSH_USERNAME`       |                          |
| `eventTime`    | `BK_CI_REPO_GIT_WEBHOOK_EVENT_TIME`          | 空则脚本省略该字段                |

`stepId` 留空由蓝盾生成。webhook 变量名落地时按工蜂触发器实测校正。

### 触发器同步写入格式（待实现）

本期无代码。后续按策略对 trigger stage 的 `codeGitWebHookTrigger` **合并式**增删改，禁止整模板 `UpdatePipeline`。Ensure 模板
`elements` 为空，不放占位触发器。一策略对应一元素：

| 字段                                                 | 约定                                                             |
|----------------------------------------------------|----------------------------------------------------------------|
| `eventType` / `version`                            | `PUSH` / `2.*`                                                 |
| `repositoryType` / `repositoryHashId`              | 按应用代码库；推荐 `ID` + hashId                                        |
| `branchName` / `excludeBranchName`                 | 由 `branchMatchMode` / `branchMatchValue` 映射                    |
| `includePaths` / `excludePaths` / `pathFilterType` | 由 `pathFilter` 映射；前缀 `NamePrefixFilter`，通配 `RegexBasedFilter`  |
| `includePushAction`                                | 如 `["push-file", "new-branch"]`                                |
| `customVariables`                                  | 至少 `BKMS_TRIGGER_POLICY_ID=<policyID>`；可选 `BKMS_TRIGGER_EVENT` |
| `stepId`                                           | 留空，**不得**写入 policyID                                           |
| `name`                                             | 可读，便于蓝盾 UI 区分策略                                                |

### 错误码

`bkerrs.GinErrorOutput`。本地 `binding` → `INVALID_REQUEST`；查库 / 跨字段 → `INVALID_ARGUMENT`。

| HTTP | ErrCode                 | 场景                                                                                                      |
|------|-------------------------|---------------------------------------------------------------------------------------------------------|
| 400  | `INVALID_REQUEST`       | 名称长度、匹配方式等 gin binding                                                                                  |
| 400  | `INVALID_ARGUMENT`      | 重名、超 5 条、硬冲突、未开自动 tag、非源码/非工蜂、匹配值与 mode 不符、存在策略时改 `repoAlias` / `CodeRepo.Type` / `sourceType` 或关自动 tag |
| 401  | `UNAUTHENTICATED`       | 回调凭证缺失、错误或与 appID 不匹配                                                                                   |
| 403  | `NO_PERMISSION`         | 写无构建权限，或列表 / 预检无查看权限                                                                                    |
| 404  | `NOT_FOUND`             | 策略不存在                                                                                                   |
| 429  | `RATE_LIMIT_EXCEEDED`   | 单应用回调 > 60 次/分钟                                                                                         |
| 500  | `INTERNAL_SERVER_ERROR` | 构建配置失效或蓝盾报错                                                                                             |

## 安全与生命周期

- 回调凭证明文不出现在任何响应（含 `details`）
- 回调只用应用独享凭证，不再校验用户权限；权限卡在策略写操作
- 自动构建由 bkms 公共账号执行，构建记录触发人展示 `--`

| 能力        | 时机                                  | 失败                                  |
|-----------|-------------------------------------|-------------------------------------|
| `Ensure`  | 首条策略创建前（`PolicyManager` 调用）         | 策略不落库；已建凭证回收，远程流水线若已创建也删掉           |
| `Cleanup` | 末条删除前（`List` 长度为 1 时先 Cleanup 再删策略） | 策略仍在。删流水线失败则保留本地并报错；凭证回收失败则告警并继续清本地 |

第 2 条起不重复 `Ensure`，复用已有 `pipelineID`。本期不同步工蜂触发器（调用点留 `FIXME`），`triggerID` 留空。`Cleanup` 不查
`PolicyStore`，是否末条由 `PolicyManager` 判定。

存在任意策略时，`UpdateBuildConfig` 拒绝改 `repoAlias`、`CodeRepo.Type`、`sourceType`、关闭自动 tag；`repoURL` / 默认分支 /
`semver`↔`custom` 放行。提示：「存在自动触发策略时不可修改，请先删除全部策略」。
