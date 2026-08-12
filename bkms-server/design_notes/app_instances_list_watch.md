# 应用实例 List 全量与 Watch 契约

本文档固化实例列表「List 全量首包 + Watch 投影增量」的接口契约，供后续实现卡对齐。
**不含** List 全量拉取、Watch 代理推送或前端改造的业务实现细节。

## 1. 背景

UI 将从「分页 List + 定时轮询」改为「一次 List 全量 + Watch 增量」。CLI / 非 UI 调用方继续使用分页 List。
服务端筛排（keyword / status / orderBy / order）与 MongoDB Pod 缓存方案已废弃。

## 2. List：可选全量参数 `all`

```http
GET /bkms/v1/bkms-server/apps/{appID}/envs/{envName}/instances
  ?page=1&pageSize=5&trafficLaneName=xxx&all=true
```

| 规则                     | 说明                                                                         |
|------------------------|----------------------------------------------------------------------------|
| 未传 `all` 或 `all=false` | 分页模式，行为与改造前一致（CLI）                                                         |
| `all=true`             | 语义：一次返回该应用环境下匹配的全部实例投影                                                     |
| 与分页共存                  | `all=true` 与 `page` / `pageSize` 同时出现时**忽略分页**，不以 400 拒绝                   |
| 响应形态                   | `{ count, results: AppInstanceOutputObj[] }`；全量实现后 `count == len(results)` |
| 投影字段                   | 复用既有 `AppInstanceOutputObj`（含 `polarisInfos`），不增删字段                        |
| 权限                     | 与现有 List 相同：目标应用在目标环境的查看权限                                                 |

**本阶段状态**：查询参数与 OpenAPI 已声明；全量拉取业务由后续需求（N2）实现。当前即便传入 `all=true`，运行时仍走现状分页。

## 3. Watch：投影事件流

```http
GET|WS /bkms/v1/bkms-server/apps/{appID}/envs/{envName}/instances/watch?trafficLaneName=
```

路径与 List 同级，须在路由表中注册在 `:instanceID` 通配之前。

### 3.1 事件逻辑结构

```json
{
  "type": "ADDED | MODIFIED | DELETED",
  "object": {
    /* AppInstanceOutputObj；DELETED 可仅含 id 等定位字段 */
  }
}
```

| 规则   | 说明                                                                 |
|------|--------------------------------------------------------------------|
| 事件类型 | `ADDED` / `MODIFIED` / `DELETED`                                   |
| 投影   | 平台投影，对齐 `AppInstanceOutputObj`（含 `polarisInfos`），**不是**原生 Pod JSON |
| 权限   | 与 List 同级；鉴权失败不得建立匿名流                                              |
| 作用域  | 按 app + env（+ 可选泳道）订阅该部署 LabelSelector 匹配范围内的实例                    |

### 3.2 传输形态（待定）

**Q-N1-001**：最终选用 SSE 或 WebSocket，在技术方案评审前不定稿。
OpenAPI 当前以 `GET` + 事件 JSON 模型描述逻辑结构；编码格式随传输形态而定。

**本阶段状态**：路由、查询参数、事件模型与 OpenAPI 已声明；handler 返回 `NOT_IMPLEMENTED`。推送代理由后续需求（N3）实现。

## 4. 承接关系

| 能力                         | 承接             |
|----------------------------|----------------|
| List `all=true` 全量拉取与北极星合并 | N2             |
| Watch 代理集群变更并推送投影事件        | N3             |
| 前端 List+Watch、去轮询、本地筛排     | 后续前端卡          |
| 传输形态 SSE / WS 选型           | 技术方案（Q-N1-001） |

## 5. 代码锚点

- Watch 领域事件类型：`pkg/workload/instance/watch/types.go`（`EventType`）
- List / Watch 查询与 API DTO：`pkg/workload/instance/serializer/instance.go`
- List handler：`pkg/workload/instance/handler/instance.go`（`ListAppInstances`）
- Watch stub：`pkg/workload/instance/handler/watch.go`（`WatchAppInstances`）
- 路由：`pkg/workload/instance/router.go`
