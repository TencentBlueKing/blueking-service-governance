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
| `versionRule.type`        | enum   | `custom` / `semver`                        |
| `versionRule.prefix`      | string | 自定义前缀，长度 ≤ 16，由字母、数字、`-` 组成，需满足容器镜像 tag 规范 |
| `versionRule.withBranch`  | bool   | 版本号是否拼接分支名                                 |
| `status`                  | enum   | `enabled` / `disabled`                     |
| `pipelineID`              | string | 关联的蓝盾触发专用流水线 ID                            |
| `triggerID`               | string | 关联的蓝盾触发器标识                                 |
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

**遗留改造项（不在 W1 范围）**：`isBuiltinPipelineType` 目前用 `slices.Contains` 精确匹配 `builtinPipelineTypes`。触发专用流水线的复合
type 匹配不上，会走进「用户自定义流水线」分支把 `pipelineID` 直接设为
type，而触发专用流水线是需要靠模板创建的。因此「触发专用流水线按应用下发」子需求落地时，必须把该判定改为前缀匹配，并从复合
type 中解析出用于查模板的类型。

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
  "pathFilter": "src/**",
  "versionRule": {
    "type": "custom",
    "prefix": "dev",
    "withBranch": true
  }
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

触发专用流水线中的 bash 脚本从蓝盾内置变量取值后拼装请求体。字段名与变量的对应关系在此固化，避免两侧对不上：

| 请求体字段          | 蓝盾流水线变量                                      | 说明                 |
|----------------|----------------------------------------------|--------------------|
| `appID`        | 流水线自定义变量                                     | 路径参数，流水线初始化时注入固定值  |
| `policyID`     | 流水线自定义变量                                     | 每个触发器对应一条策略，初始化时注入 |
| `event`        | —                                            | 固定字面量 `push`       |
| `branch`       | `BK_CI_REPO_GIT_WEBHOOK_BRANCH`              | 推送的分支名             |
| `commitID`     | `BK_CI_REPO_GIT_WEBHOOK_PUSH_HEAD_COMMIT_ID` | 本次推送的 HEAD commit  |
| `commitAuthor` | `BK_CI_REPO_GIT_WEBHOOK_PUSH_USERNAME`       | 推送人                |
| `eventTime`    | `BK_CI_REPO_GIT_WEBHOOK_EVENT_TIME`          | 事件时间               |

上表变量名以蓝盾工蜂 Git 事件触发器的实际输出为准，「触发专用流水线按应用下发」子需求落地时需实测校正。

### 错误码

沿用 `pkg/common/bkerrs` 的 `ErrCode` 枚举，响应体为 `bkerrs.GinErrorOutput`。

| HTTP | ErrCode                 | 触发场景                                         |
|------|-------------------------|----------------------------------------------|
| 400  | `INVALID_REQUEST`       | 参数格式不合法：名称长度越界、匹配方式非法、匹配值缺失                  |
| 400  | `INVALID_ARGUMENT`      | 参数格式合法但不满足业务规则：名称重复、超出策略数量上限、硬冲突、存在策略时修改构建配置 |
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
