# 实例监控：前端接入文档

> 后端分支：`bkms-server/bkm/fix-instance-time-series-api`，story `139716566`

## 1. 整体流程

和蓝鲸监控的容器页面一样，分两步：

1. **选实例**：调 Top N 接口（`GET .../top-instances`），按平均 CPU 使用量拿到前 N 个 Pod 名；
2. **查曲线**：把这批 Pod 名作为 `instances` 传给时序接口（`POST .../instance-time-series`），每个指标请求一次。

后端不再枚举 Pod，也不再有 `all=true`，大实例数环境（约 2000 个 Pod）不会再把服务内存打满。

## 2. 实例 Top N 接口（新增）

```
GET /api/bkms/v1/bkms-server/apps/{appID}/envs/{envName}/bkmonitor/top-instances
```

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `startTime` | number | 是 | 排名时间范围的开始时间，Unix **秒**，必须 ≤ `endTime` |
| `endTime` | number | 是 | 排名时间范围的结束时间，Unix **秒** |
| `limit` | number | 否 | 取前 N 个，1–50，默认 10 |
| `limitType` | `top` \| `bottom` | 否 | 默认 `top`：从高到低；`bottom`：从低到高 |
| `interval` | number | 否 | 排名子查询的步长（秒），最小 60，默认 60，建议与图表的汇聚周期一致 |
| `method` | `sum` \| `max` \| `min` \| `count` \| `avg` | 否 | 同一 Pod 内多个容器的汇聚方法，默认 `sum`，建议与图表的汇聚方法一致 |

**排名口径**：时间范围内的**平均 CPU 使用量（cores）**。候选实例是该环境默认泳道最新一次部署的主工作负载（GameDeployment / Deployment）下的全部 Pod。

**响应**（已按 `limitType` 排好序）：

```json
{
  "data": [
    { "instance": "demo-0", "value": 0.83 },
    { "instance": "demo-3", "value": 0.61 }
  ]
}
```

- `value` 是平均 CPU 使用量，单位核数，仅供展示和调试，排序以数组顺序为准。
- 时间范围内没有监控数据时返回 `data: []`，此时不要再请求时序接口（`instances` 为空会返回 400）。

**错误**：

| HTTP | 场景 |
|---|---|
| 400 | 参数非法；应用不是 AppModel 类型；工作空间未开通蓝鲸监控；环境未绑定集群或命名空间 |
| 404 | 应用或环境不存在；该环境还没有部署记录 |
| 500 | 查询蓝鲸监控失败 |

## 3. 时序接口（精简）

```
POST /api/bkms/v1/bkms-server/apps/{appID}/envs/{envName}/bkmonitor/instance-time-series
Content-Type: application/json
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `instances` | string[] | 是 | 实例名，过滤空白后 1–50 个；后端**不校验**实例是否属于该环境 |
| `metricKey` | string | 是 | 指标标识，见下表 |
| `startTime` / `endTime` | number | 是 | Unix 秒，`startTime ≤ endTime` |
| `interval` | number | 否 | 汇聚周期（秒），小于 60 按 60 处理 |
| `method` | `sum` \| `max` \| `min` \| `count` \| `avg` | 否 | 汇聚方法，默认 `sum`。使用率类指标的分子分母用同一方法，`count` 下使用率恒为 1 |

**已删除的字段**：`all`、`limit`、`limitType`。排名统一由 Top N 接口负责，时序接口按传入的实例原样返回、不排序也不截断。

| metricKey | 含义 | unit |
|---|---|---|
| `cpu_usage` | CPU 使用量 | `cores` |
| `cpu_request_usage` | CPU requests 使用率 | `percent`（0–1，展示时 ×100） |
| `cpu_limit_usage` | CPU limits 使用率 | `percent` |
| `memory_usage` | 内存使用量（Working Set） | `bytes` |
| `memory_request_usage` | 内存 requests 使用率 | `percent` |
| `memory_limit_usage` | 内存 limits 使用率 | `percent` |
| `network_receive` | 网络入带宽 | `bytes/s` |
| `network_transmit` | 网络出带宽 | `bytes/s` |
| `disk_usage` | 磁盘使用量 | `bytes` |

响应结构不变：`Record<metricKey, { displayName, unit, series: [{ instance, dataPoints, stat }] }>`，`dataPoints` 每个元素是 `[值, 毫秒时间戳]`，统计值在 `stat.xxx[1]`。

## 4. SDK

```bash
cd bkms-ui && pnpm gen:api:v1
```

```ts
// 1. 选实例
const top = await BkintegrationsBkmonitorService.listTopInstances({
  appID, envName, startTime, endTime, interval, method, limit: 10, limitType: 'top',
});
const instances = top.map(i => i.instance);

// 2. 查曲线（每个指标一次）
if (instances.length) {
  const data = await BkintegrationsBkmonitorService.getInstanceTimeSeries({
    appID, envName, instances, metricKey, startTime, endTime, interval, method,
  });
}
```

`fetch.ts` 的 `parseUrlAndParams` 会从传入对象上 `delete` 掉 `appID`、`envName`，每次请求请传新的对象字面量。

## 5. 交互对接

- **从顶部「监控」进入**：先调 Top N 拿实例，下拉框占位显示「全部实例（N）」，N 取实例列表接口的 `count`。
- **Limit top / bottom**：变更时重新调 Top N，再用新名单重查所有图表。
- **汇聚周期、汇聚方法、时间范围变更**：Top N 和时序都要用新参数重查。
- **手动选实例**：跳过 Top N，直接用选中的实例（最多 50 个）查时序。
- **从实例行进入**：`instances: [instanceId]`，跳过 Top N。
- **汇总表**：行就是 Top N 返回的那批实例，顺序和 Top N 一致；各列用同一批实例从对应指标的时序结果里取 `stat.last[1]`。
- **时间对比**：对比时段也用当前这批实例查时序，不要重新排名，否则两个时段的实例对不上。
- **多环境**：每个环境各调一次 Top N（`limit` 相同），前端合并后按 `value` 再排一次取前 N（top 降序、bottom 升序），再按环境分组查时序。
- **请求去重**：`monitor-chart-bridge.ts` 的 `dedupKey` 要包含 `instances`、`method`、`interval`。

## 6. 排名口径说明

Top N 只按 CPU 使用量排一次，所有图表（CPU、内存、网络）共用这一份名单，和蓝鲸监控容器页面的做法一致。因此内存、网络图展示的是「CPU 前 N 的 Pod 的内存 / 网络」，不是各自指标的前 N。
