# 应用实例 List 全量与 Watch 契约

本文档固化实例列表「List 全量首包 + Watch 投影增量」的接口契约，供后续实现卡对齐。
**不含** Watch 代理推送或前端改造的业务实现细节。

## 1. 背景

UI 将从「分页 List + 定时轮询」改为「一次 List 全量 + Watch 增量」。CLI / 非 UI 调用方继续使用分页 List。
服务端筛排（keyword / status / orderBy / order）与 MongoDB Pod 缓存方案已废弃。

## 2. List：可选全量参数 `all`

```http
GET /bkms/v1/bkms-server/apps/{appID}/envs/{envName}/instances
  ?trafficLaneName=xxx
  &all=true                         # 全量：禁止同时带 page / pageSize
  &page=1&pageSize=5                # 分页：不传 all 时必填
```

| 规则                     | 说明                                                                   |
|------------------------|----------------------------------------------------------------------|
| 未传 `all` 或 `all=false` | 分页模式；`page` / `pageSize` 必填；当前页解析失败则整次失败（与改造前一致，供 CLI 使用）            |
| `all=true`             | 一次返回该应用环境下匹配的全部可投影实例；单个 Pod 无法投影时跳过                                  |
| 与分页互斥                  | `all=true` 与 `page` 或 `pageSize` 任一同时出现时返回参数错误（400），不返回数据            |
| 响应形态                   | `{ count, results, skippedCount, skipped }`                          |
| `count`                | 全量：`count == len(results)`（不含跳过项）；分页：LabelSelector 匹配的 Pod 总数        |
| `skipped*`             | 仅全量模式写入无法投影的实例；分页无跳过（解析失败即 500）；无跳过时 `skippedCount=0` 且 `skipped=[]` |
| 投影字段                   | 复用既有 `AppInstanceOutputObj`（含 `polarisInfos`），不增删展示字段                |
| 北极星                    | 拉取失败不阻塞 Pod 输出：分页与全量均降级为 `polarisInfos=[]`，见下                        |
| 权限                     | 与现有 List 相同：目标应用在目标环境的查看权限                                           |

北极星是旁路信息源，它不可用不应该牵连 Pod 数据，因此拉取失败时不再整次 500，而是降级为 `polarisInfos=[]`，
K8s 字段照常返回，失败原因只落 WARN 日志。降级结果与「该应用确实没注册北极星」同形，服务端不做区分，前端一律展示为未知。

**本阶段状态**：N2 已落地全量拉取、互斥校验、全量投影失败跳过与北极星合并。

## 3. Watch：投影事件流

```http
GET|WS /bkms/v1/bkms-server/apps/{appID}/envs/{envName}/instances/watch?trafficLaneName=
```

路径与 List 同级，须在路由表中注册在 `:instanceID` 通配之前。

### 3.1 事件逻辑结构

```json5
{
  "type": "ADDED | MODIFIED | DELETED",
  "object": {}
  // AppInstanceOutputObj；DELETED 可仅含 id 等定位字段
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

| 能力                                       | 承接             |
|------------------------------------------|----------------|
| List `all=true` 全量拉取、互斥校验、北极星合并与 skipped | N2（已落地）        |
| Watch 代理集群变更并推送投影事件                      | N3             |
| 前端 List+Watch、去轮询、本地筛排                   | 后续前端卡          |
| 传输形态 SSE / WS 选型                         | 技术方案（Q-N1-001） |

## 5. 代码锚点

- Watch 领域事件类型：`pkg/workload/instance/watch/types.go`（`EventType`）
- List / Watch 查询与 API DTO：`pkg/workload/instance/serializer/instance.go`
- List handler：`pkg/workload/instance/handler/instance.go`（`ListAppInstances` 编排）、`list.go`（绑定 / 部署记录 / 投影 /
  北极星）
- Watch stub：`pkg/workload/instance/handler/watch.go`（`WatchAppInstances`）
- 路由：`pkg/workload/instance/router.go`
