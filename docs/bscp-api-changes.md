# BSCP 接入新版本 — 前端 API 变更说明

- 提交：`4f7c10a4 feat: 接入新版本BSCP --story=137794713 (#174)`
- 对外基础路径：`/bkms/v1/bkms-server`
- 成功响应直接返回业务对象（无 `result` 包装）；失败返回 `{"error": {code, message, system, module, details}}`，HTTP 400/404。

## 一、接口变更总览

| Method | Path | 说明 | 变化 |
| --- | --- | --- | --- |
| POST | `/apps/{appID}/bscpcfg/metadata` | 初始化配置管理（无 body，幂等） | 接口不变；新增前置校验：workspace 必须已绑定 BSCP 项目 |
| GET | `/apps/{appID}/bscpcfg/metadata` | 获取元信息 | 接口不变；响应字段变化 |
| PATCH | `/apps/{appID}/bscpcfg/metadata` | 修改元信息 | 请求体不变；响应字段变化 |
| DELETE | `/apps/{appID}/bscpcfg/metadata` | 删除元信息（级联环境绑定） | 无变化 |
| GET | `/apps/{appID}/bscpcfg/envs` | 环境绑定列表 | 接口不变；响应字段变化 |
| POST | `/apps/{appID}/bscpcfg/envs/{envName}/binding` | 创建环境绑定（无 body） | 请求不变；响应字段变化 |
| GET | `/apps/{appID}/bscpcfg/envs/{envName}/binding` | 环境绑定详情 | 接口不变；响应字段变化 |
| DELETE | `/apps/{appID}/bscpcfg/envs/{envName}/binding` | 删除环境绑定 | 无变化 |
| ~~PATCH~~ | ~~`/apps/{appID}/bscpcfg/envs/{envName}/binding`~~ | ~~更新绑定服务列表~~ | **已删除** |

## 二、请求 / 响应

### 1. Metadata

`PATCH /apps/{appID}/bscpcfg/metadata` 请求体（字段可选，至少一个）：

```json
{
  "mountPath": "/data/bscp",
  "workloadName": "my-app",
  "workloadKind": "GameDeploy"
}
```

`GET / POST / PATCH` 响应：

```json
{
  "data": {
    "appID": "xxx",
    "postHookID": "123",
    "mountPath": "/data/bscp",
    "workloadName": "my-app",
    "workloadKind": "GameDeploy",
    "enable": true,
    "operator": "admin",
    "createdAt": "2026-09-30T00:00:00Z",
    "updatedAt": "2026-09-30T00:00:00Z"
  }
}
```

字段变化：

- 新增：`enable`（是否启用 BSCP 配置管理）
- 删除：`bscpBizID`、`credentialName`、`feedAddr`、`token`

行为说明：

- `POST` 无请求体；workspace 未绑定 BSCP 项目时返回 404（`workspace missing bscp project`）。
- `GET` 在未初始化时返回 404（`bscpcfg metadata not found`），前端应先调 `POST`。
- `enable=false` 时不允许创建环境绑定。

### 2. EnvBinding

`POST /apps/{appID}/bscpcfg/envs/{envName}/binding` 无请求体。

`GET`（详情/列表）、`POST` 响应：

```json
{
  "data": {
    "appID": "xxx",
    "envName": "dev",
    "mountPath": "/data/bscp",
    "workloadName": "my-app",
    "workloadKind": "GameDeploy",
    "bscpEnvID": "12",
    "bscpEnvName": "dev",
    "bscpAppID": "345",
    "operator": "admin",
    "createdAt": "2026-09-30T00:00:00Z",
    "updatedAt": "2026-09-30T00:00:00Z"
  }
}
```

列表接口为 `{"data": [ ... ]}`。

字段变化：

- 新增：`bscpEnvID`、`bscpEnvName`、`bscpAppID`
- 删除：`apps`、`bscpBizID`、`feedAddr`、`token`、`defaultFileAppID`

行为说明：

- 创建绑定时后端自动完成：按 `envName` 匹配或创建 BSCP 环境 → 以 `appID` 作为 BSCP App 名称创建或复用 App → 绑定后置脚本、刷新 Credential 权限范围。
- bkms 环境类型自动映射为 BSCP 环境类型：`development → dev`、`test → test`、`staging → staging`、`production → prod`。
- 重复创建返回 409（`already exists`）。

### 3. Workspace（连带变化）

`bkSystems` 新增只读字段：

| 字段 | 说明 |
| --- | --- |
| `bkBSCPProjectID` | BSCP 项目 ID |
| `bkBSCPProjectKey` | BSCP 项目 Key / Code（如 `BK-BSCP-00012`） |
| `bscpCredentialID` | BSCP Credential ID |

Token 不对外输出。

## 三、前端改动清单

1. 移除对 `PATCH .../envs/{envName}/binding` 的调用（“编辑绑定服务列表”功能下线）。
2. 重新生成类型：`pnpm gen:api:v1`。
3. 字段替换：
   - `apps` → `bscpAppID`
   - `bscpBizID`、`defaultFileAppID` → `bscpEnvID` / `bscpEnvName` / `bscpAppID`
   - `feedAddr`、`token`、`credentialName` → 不再返回，删除相关展示
   - Metadata 新增 `enable`，可据此控制开关态展示
4. 创建绑定不再需要提交服务列表，按钮直接 `POST` 即可。

## 四、BSCP 接入基础说明

**BSCP 元信息从 Workspace 接口获取**，不在 bscpcfg 接口里：

```
GET /apps/{appID}                      → 取 workspaceID
GET /workspaces/{workspaceID}          → 取 data.bkSystems
```

```json
{
  "data": {
    "id": "ws-xxx",
    "bkSystems": {
      "bkCCBizID": "2",
      "bkBSCPProjectID": "3",
      "bkBSCPProjectKey": "BK-BSCP-00012",
      "bscpCredentialID": "18"
    }
  }
}
```

其他约定：

- Workspace 创建时后端自动绑定 BSCP 的 Default 项目，并创建固定凭据 `bkms-credential`；无需前端触发。
- 层级关系：BSCP 项目（Project）→ 环境（Environment）→ App。
- 一个 bkms 应用对应一个 BSCP App，App 名称即 bkms `appID`。
- 一个 bkms 应用环境对应一个 BSCP 环境，按 `envName` 匹配或创建；环境类型自动映射：
  `development → dev`、`test → test`、`staging → staging`、`production → prod`。
- 创建环境绑定后，后端会自动完成 BSCP 环境/App 创建、后置脚本绑定和凭据权限刷新。
