# 自动触发镜像构建：接口契约与数据模型

## 设计背景与目标

父需求「自动触发镜像构建」跨 bkms-server 与 bkms-ui 两端，涉及触发策略管理、触发专用流水线下发、回调处理、触发记录四条链路。若各子需求各自定义数据结构，联调时必然出现字段对不上的返工。

本文档是该需求的**唯一契约事实源**：把跨子需求的数据模型与接口契约一次性定死，使各子需求从「依赖别人的实现」降级为「依赖同一份契约」，从而并行开工。

本文档对应的落码范围（W1）只包含：数据模型结构体、Store 接口定义、Serializer 定义、索引迁移文件、以及只做参数绑定校验的空实现
Handler。**不含任何业务实现**。

### 接口实现归属与收敛

W1 已注册全部 8 条路由，但 Handler 均为空实现，返回契约结构的零值。这意味着在 Wave 2 落地前，这批接口可调用但永远返回空数据。各接口的真实实现归属如下：

| 接口                              | 实现归属子需求                  |
|---------------------------------|--------------------------|
| 策略列表 / 创建 / 更新 / 启停 / 删除 / 冲突预检 | 触发策略后端管理与冲突检测            |
| 触发记录查询                          | 自动触发记录存储与查询              |
| 构建回调                            | 构建回调处理与版本号生成、构建回调凭证鉴权与限流 |

## 数据模型

### 触发策略 `build_trigger_policies`

对应 Go 结构体 `pkg/build/trigger.Policy`。

| 字段                        | 类型     | 约束与说明                                      |
|---------------------------|--------|--------------------------------------------|
| `id`                      | string | 策略 ID，全局唯一                                 |
| `appID`                   | string | 所属应用                                       |
| `name`                    | string | 策略名称，应用内唯一，长度 1–32，由汉字、大小写字母、数字、`-`、`_` 组成 |
| `event`                   | enum   | 触发事件，本期唯一取值 `push`                         |
| `branchMatchMode`         | enum   | `eq` / `prefix` / `all`                    |
| `branchMatchValue`        | string | 分支匹配值，多值以英文逗号分隔；`all` 时为空                  |
| `pathFilter`              | string | 文件路径条件，留空表示全匹配                             |
| `status`                  | enum   | `enabled` / `disabled`                     |
| `pipelineID`              | string | 关联的蓝盾触发专用流水线 ID                            |
| `triggerID`               | string | 关联的蓝盾触发器元素标识，供触发器同步增删改时定位               |

镜像 tag 规则**不落在策略上**，统一使用应用 `buildConfig.tagConfig`（与手动构建推荐版本号同源）。创建 / 更新策略时须校验 `tagConfig.IsAutoGenerateEnabled()`（`type` 为 `semver` 或 `custom`）；未开启则拒绝，错误码 `INVALID_ARGUMENT`。自动触发回调生成版本号时同样读取该配置。
| `creator`                 | string | 创建人                                        |
| `createdAt` / `updatedAt` | time   | 审计字段                                       |

索引：唯一 `appID + name`；普通 `appID`。

数量约束：单应用策略上限 5 条，生效中与已停用合并计数。该约束由业务层校验，不由索引保证。

### 触发记录 `build_trigger_records`

对应 Go 结构体 `pkg/build/trigger.Record`。

| 字段             | 类型     | 约束与说明                          |
|----------------|--------|--------------------------------|
| `policyID`     | string | 归属策略                           |
| `appID`        | string | 归属应用                           |
| `triggeredAt`  | time   | 触发时间                           |
| `event`        | enum   | 事件类型，与策略同枚举                    |
| `branch`       | string | 分支名                            |
| `commitID`     | string | commit 哈希                      |
| `commitAuthor` | string | commit 作者                      |
| `result`       | enum   | `built` / `skipped` / `failed` |
| `buildID`      | string | 结果为 `built` 时关联的构建号，其余为空       |
| `reason`       | string | 跳过或失败原因，`built` 时为空            |
| `createdAt`    | time   | 创建时间                           |

索引：普通 `policyID + triggeredAt`；普通 `policyID + commitID`。

`policyID + commitID` **不能设唯一约束**。去重规则是「同一策略下同一 commit 已成功构建过则跳过」，同一 commit 完全可能先产生若干条
`skipped` 或 `failed` 记录，唯一约束会让这些记录写不进去。去重必须由业务层按 `result == built` 查询判定。

触发记录本期全量落库长期保留，不做条数上限、过期清理与归档。

### 构建记录扩展

在现有 `pkg/build/image.Record`（集合 `build_records`）上追加两个字段：

| 字段                | 类型     | 说明                   |
|-------------------|--------|----------------------|
| `triggerType`     | enum   | `manual` / `auto`    |
| `triggerPolicyID` | string | 自动触发时关联的策略 ID，手动触发为空 |

### 触发专用流水线的存储口径

本需求新增一类**触发专用流水线**，内容仅为「工蜂 Git 事件触发器 + 回调 bkms 的脚本步骤」，自身不执行构建。它与现有共享构建流水线并存，两者唯一性口径不同：

| 流水线      | `type` 取值                       | 唯一性       |
|----------|---------------------------------|-----------|
| 共享构建流水线  | `dockerfile` / `helm-git-build` | 工作空间级唯一   |
| 用户自定义流水线 | 流水线 ID（`p-[a-z0-9]{32}`）        | 工作空间级唯一   |
| 触发专用流水线  | `build-trigger-{appID}`         | **应用级唯一** |

`bkci_pipelines` 的唯一索引保持 `workspaceID + type` 不变，不做任何索引变更。应用级唯一是通过**把 appID 编码进 `type`**
达成的：`workspaceID + build-trigger-{appID}` 天然等价于 `workspaceID + appID + 触发专用类型`。

这个做法沿用了 `type` 字段已有的语义——它本来就不是纯枚举，用户自定义流水线的 `type` 直接就是 pipelineID。编码与解析函数见
`pkg/bkintegrations/bkci`。

**W2 已落地**：

- `isBuiltinPipelineType` / `ResolveBuiltinTemplateType`：共享类型精确匹配，触发专用复合 type 前缀匹配后解析模板类型 `build-trigger`
- 模板资产：`assets/pipeline_templates/build_trigger.json`（trigger stage 的 `elements: []` + 回调脚本；Git 触发器由同步下发填充）
- 模板渲染：与构建镜像相同走 `[[ ]]` / `renderPipelineTemplate`；Reload 落地 `builderImageCode`；
  实例期字段（`appID` / `callbackURL` / `credentialID`）在资产中自逃逸，Reload 后保留占位，
  `TriggerPipelineManager.Ensure` 二次渲染显示名、回调脚本等
- 内部 API（无新 REST）：`NewTriggerPipelineManager(workspaceID).Ensure/Cleanup(ctx, appID)`
  （`pkg/bkintegrations/bkci`）
- 蓝盾侧显示名 / 描述：来自模板 `name` / `description`（name 含 `[[ .appID ]]`）
- 回调地址：`{httpServer.publicBaseURL}/bkms/v1/bkms-server/apps/{appID}/build-trigger-policies/callback`
- 回调凭证：蓝盾 `ACCESSTOKEN` 类型，每应用一条；本地 `bkci_pipelines.callbackCredentialID` 记录凭证 ID；凭证明文不回显
- `Initialize(build-trigger-*)`：已存在则返回；不存在则拒绝创建（必须走 Ensure，避免无凭证注入）
- **不做模板 semver 自动升级**：`Ensure` / `Initialize` 对已存在实例均 create-if-missing / 原样返回。共享流水线的整模板
  `UpdatePipeline` 会冲掉带 appID 的显示名、已注入的回调脚本，以及触发器同步写入的 Git 条件；这不是循环依赖，而是多写入方共存下不能复用该升级路径。若需滚动模板，应另做保留上述字段的合并式 Sync

## 接口契约

全部接口挂在服务公共前缀 `/bkms/v1/bkms-server` 下。下表路径省略该前缀。

### 接口签名

| 接口     | 方法与路径                                                   | 鉴权                | 出参             |
|--------|---------------------------------------------------------|-------------------|----------------|
| 策略列表   | GET `/apps/{appID}/build-trigger-policies`                      | 用户票据 + 查看权限       | 全部策略 + 总数（不分页，上限 5） |
| 创建策略   | POST `/apps/{appID}/build-trigger-policies`                     | 用户票据 + 构建权限       | 策略实体           |
| 更新策略   | PUT `/apps/{appID}/build-trigger-policies/{policyID}`          | 用户票据 + 构建权限       | 策略实体           |
| 启停策略   | PATCH `/apps/{appID}/build-trigger-policies/{policyID}/status` | 用户票据 + 构建权限       | 策略实体           |
| 删除策略   | DELETE `/apps/{appID}/build-trigger-policies/{policyID}`       | 用户票据 + 构建权限       | 204 无内容        |
| 冲突预检   | POST `/apps/{appID}/build-trigger-policies/conflict-check`      | 用户票据 + 查看权限       | 冲突级别 + 冲突策略名列表 |
| 触发记录查询 | GET `/apps/{appID}/build-trigger-policies/{policyID}/records`  | 用户票据 + 查看权限       | 记录列表 + 总数      |
| 构建回调   | POST `/apps/{appID}/build-trigger-policies/callback`            | **应用独享凭证**，不走用户票据 | 三态处理结果         |

回调接口是本项目第一个面向外部系统的路由，它注册在不带 `auth.Required` 的路由组上。除它以外，其余 7 个接口均走现有用户票据鉴权，并在
handler 内用 `perm.ValidateAppByID` 做应用级权限校验。

`conflict-check` 与 `callback` 这两个静态路径段和同级的 `{policyID}` 路径参数共存。gin 1.12 支持这种混用，现有
`/apps/auto-id-suffix` 与 `/apps/{appID}` 即为先例。

### 请求与响应结构

响应统一外层包 `data`。策略列表一次返回全部（上限 5 条，不分页），仍用 `count` + `results`；触发记录等可能无限增长的列表才做分页，分页字段统一用 `count` + `results`。

策略表单（创建、更新、冲突预检共用）：

```json
{
  "name": "dev-auto-build",
  "event": "push",
  "branchMatchMode": "prefix",
  "branchMatchValue": "feature/,hotfix/",
  "pathFilter": "src/**"
}
```

冲突预检额外接受 `excludeTriggerID`，用于编辑场景排除自身。其响应：

```json
{
  "data": {
    "level": "warn",
    "conflictPolicyNames": [
      "prod-build"
    ]
  }
}
```

`level` 取值 `none` / `warn` / `error`。`error` 为硬冲突，前端必须禁止保存；`warn` 为软冲突，允许保存但需告警。

触发记录查询支持 `result` 筛选（`built` / `skipped` / `failed`，留空为不筛选）与 `page` / `pageSize` 分页。

回调请求体与响应：

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

三态均以 HTTP 200 返回，由 `result` 字段区分，使流水线侧脚本能从响应体判断处理结果并留痕。`built` 时 `buildID` 非空、
`reason` 为空；`skipped` 与 `failed` 时 `buildID` 为空、`reason` 说明原因。

### 回调字段与蓝盾流水线变量映射

触发专用流水线中的 bash 脚本从蓝盾变量取值后拼装请求体。字段名与变量的对应关系在此固化，避免两侧对不上：

| 请求体字段          | 来源                                                                 | 说明                                      |
|----------------|--------------------------------------------------------------------|-----------------------------------------|
| `appID`        | 路径参数                                                               | Ensure 写入回调 URL，不进 request body         |
| `policyID`     | 命中触发器 `additionalOptions.customVariables` 的 `BKMS_TRIGGER_POLICY_ID` | 每策略一个触发器元素；由触发器同步写入，**不用** `stepId` 承载 |
| `event`        | 同上路径的 `BKMS_TRIGGER_EVENT`，缺省 `push`                                 | 预留扩展其它触发事件                              |
| `branch`       | `BK_CI_REPO_GIT_WEBHOOK_BRANCH`                                    | 推送的分支名                                  |
| `commitID`     | `BK_CI_REPO_GIT_WEBHOOK_PUSH_HEAD_COMMIT_ID`                       | 本次推送的 HEAD commit                       |
| `commitAuthor` | `BK_CI_REPO_GIT_WEBHOOK_PUSH_USERNAME`                             | 推送人                                     |
| `eventTime`    | `BK_CI_REPO_GIT_WEBHOOK_EVENT_TIME`                                 | 事件时间；空则脚本省略该字段，避免 Go `time.Time` 解空串失败 |

`additionalOptions.customVariables` 是蓝盾插件通用的流程控制字段（非 Git 触发器专属业务表单项），触发命中后注入构建环境，供回调脚本读取。`stepId` 留空，由蓝盾创建时自动生成。

上表 webhook 变量名以蓝盾工蜂 Git 事件触发器的实际输出为准，落地时需实测校正。

### 触发器同步写入格式（待实现）

本期**无代码实现**触发器同步；以下格式供后续子需求按策略对 trigger stage 内 `codeGitWebHookTrigger` 做合并式增删改时对齐。禁止整模板 `UpdatePipeline` 覆盖（会冲掉 Ensure 注入的显示名、回调脚本与已同步的触发器）。

Ensure 下发的模板中 trigger `elements` 为空数组；首条及后续策略创建 / 更新 / 删除时，由同步向该数组增删改真实触发器，**不在模板里放未配置完的占位触发器**。

约定：**每个触发策略对应一个** `codeGitWebHookTrigger` 元素。关键字段（仓库 / 分支 / 路径等由同步填充）：

| 字段 | 约定 |
|------|------|
| `eventType` | 本期 `PUSH` |
| `version` | `2.*`（与现网插件一致） |
| `repositoryType` / `repositoryHashId` | 按应用构建配置的代码库填充；推荐 `ID` + hashId |
| `branchName` / `excludeBranchName` | 由策略 `branchMatchMode` / `branchMatchValue` 映射 |
| `includePaths` / `excludePaths` / `pathFilterType` | 由策略 `pathFilter` 映射；前缀用 `NamePrefixFilter`，通配用 `RegexBasedFilter` |
| `includePushAction` | 如 `["push-file", "new-branch"]` |
| `additionalOptions.customVariables` | 至少 `{ "key": "BKMS_TRIGGER_POLICY_ID", "value": "<policyID>" }`；可选 `BKMS_TRIGGER_EVENT` |
| `stepId` | 留空或由蓝盾生成，**不得**写入 policyID |
| `name` | 可读名称，便于在蓝盾 UI 区分策略 |

参考元素形状（值仅为示意）：

```json
{
  "@type": "codeGitWebHookTrigger",
  "atomCode": "codeGitWebHookTrigger",
  "classType": "codeGitWebHookTrigger",
  "version": "2.*",
  "name": "bkms/repo trigger for policy",
  "eventType": "PUSH",
  "repositoryType": "ID",
  "repositoryHashId": "<repoHashId>",
  "branchName": "main,release",
  "excludeBranchName": "develop,test",
  "includePaths": "/pkg",
  "excludePaths": "/log",
  "pathFilterType": "NamePrefixFilter",
  "includePushAction": ["push-file", "new-branch"],
  "stepId": "",
  "additionalOptions": {
    "enable": true,
    "enableCustomEnv": true,
    "customVariables": [
      { "key": "BKMS_TRIGGER_POLICY_ID", "value": "btp-xxx" },
      { "key": "BKMS_TRIGGER_EVENT", "value": "push" }
    ]
  }
}
```

### 错误码

沿用 `pkg/common/bkerrs` 的 `ErrCode` 枚举，响应体为 `bkerrs.GinErrorOutput`。

| HTTP | ErrCode                 | 触发场景                                         |
|------|-------------------------|----------------------------------------------|
| 400  | `INVALID_REQUEST`       | 参数格式不合法：名称长度越界、匹配方式非法、匹配值缺失                  |
| 400  | `INVALID_ARGUMENT`      | 参数格式合法但不满足业务规则：名称重复、超出策略数量上限、硬冲突、未开启自动生成 tag、存在策略时关闭自动生成 tag 或修改构建配置 |
| 401  | `UNAUTHENTICATED`       | 回调未携带凭证、凭证错误，或凭证与 appID 不匹配                  |
| 403  | `NO_PERMISSION`         | 操作人不具备该应用的构建权限                               |
| 404  | `NOT_FOUND`             | 策略 ID 无效或已被删除                                |
| 429  | `RATE_LIMIT_EXCEEDED`   | 单应用回调超过 60 次/分钟                              |
| 500  | `INTERNAL_SERVER_ERROR` | 发起构建失败：构建配置失效、蓝盾接口报错                         |

名称长度这类字段本地校验由 gin validator 的 `binding` tag 完成，因此统一落在 `INVALID_REQUEST`；需要查库或跨字段判断的校验落在
`INVALID_ARGUMENT`。

## 安全约定

- **凭证不回显**：回调凭证不出现在任何接口的响应体字段中，包括策略实体、策略列表与错误响应的 `details`。凭证由蓝盾凭证管理持有，bkms
  侧只做校验不做展示。
- **回调不做用户级鉴权**：回调接口用应用独享凭证鉴权，触发构建时不再校验用户权限。权限校验发生在策略的创建、编辑、启停环节。
- **执行身份**：自动触发的构建由 bkms 公共账号执行，构建记录的触发人对自动构建展示 `--`。

## 触发专用流水线生命周期（W2）

| 能力 | 调用时机 | 成功语义 | 失败语义 |
|------|---------|---------|---------|
| `TriggerPipelineManager.Ensure` | 应用首条触发策略创建前 | 返回已存在或新建的流水线 | 返回错误；策略不得落库；失败时回收已建凭证，且若蓝盾流水线已创建但本地落库失败则一并删除远程流水线 |
| `TriggerPipelineManager.Cleanup` | 应用最后一条触发策略删除后 | 蓝盾流水线与本地记录清理完成 | 删流水线失败则保留本地并报错；凭证回收失败则告警并继续清本地 |

Cleanup **信任调用方**已确认该应用无策略，不查询 `PolicyStore`。
