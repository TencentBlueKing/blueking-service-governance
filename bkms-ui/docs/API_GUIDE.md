# bkms-ui API 请求层指南

## 1. 架构概览

```
src/api/
├── fetch.ts           # ConsoleFetch 封装类 + 统一响应/错误处理（核心）
├── interceptors.ts    # 底层 fetch、拦截器注册器、请求 ID 与队列接入
├── clients.ts         # API 客户端实例（v1Fetch）
├── trace-id.ts        # 链路追踪 ID
├── request-queue.ts   # 请求队列（去重、路由切换取消）
├── ts-helpers.ts      # TS 类型辅助
└── modules/
    ├── user.ts            # 用户 API
    ├── bkmsserver.ts      # 旧 ApiServerService 兼容门面，委托到 v1 Service
    ├── custom.ts          # 自定义接口
    ├── rendermanager.ts   # 渲染管理
    ├── trafficmanager.ts  # 流量管理
    └── v1/                # v1 API（37 个文件，自动生成）
        ├── app.ts
        ├── deploy.ts
        ├── env.ts
        ├── index.ts
        └── ...
```

## 2. ConsoleFetch 使用

### 2.1 基本结构

`ConsoleFetch` 是基于原生 `fetch` 的封装类，提供类型安全的请求方法：

```ts
// src/api/fetch.ts
export default class ConsoleFetch {
  get<P, T>(url: string); // GET 请求
  post<P, T>(url: string); // POST 请求
  put<P, T>(url: string); // PUT 请求
  delete<P, T>(url: string); // DELETE 请求
  patch<P, T>(url: string); // PATCH 请求
}
```

- `P` — 请求参数类型
- `T` — 返回数据类型

### 2.2 API 客户端实例

```ts
// src/api/clients.ts
import Fetch from './fetch';

export const v1Prefix = import.meta.env.BK_API_V1_PREFIX || `${import.meta.env.BK_API_PREFIX}/bkms/v1/bkms-server`;

export const v1Fetch = new Fetch({
  prefix: v1Prefix,
});
```

默认配置：

```ts
{
  mode: 'cors',
  credentials: 'include',      // 携带 Cookie
  headers: {
    'X-Requested-With': 'fetch',
    'Content-Type': 'application/json',
  },
  responseType: 'json',
  validateCode: false,
  interceptorErr: true,        // 自动弹 Message
}
```

## 3. 新增 API 接口

### 3.1 自动生成（推荐）

v1 API 由 swagger.json 自动生成：

```bash
pnpm gen:api:v1
# 从 bkms-server/docs/apis/swagger.json 生成
# → src/api/modules/v1/*.ts
# → src/@types/v1/*.ts
```

**后端接口变更后重新执行此命令即可，勿手动编辑 v1/ 下文件。**

### 3.2 手动新增（非 v1 接口）

在 `src/api/modules/` 下创建文件：

```ts
// src/api/modules/my-feature.ts
import { v1Fetch } from '~/api/clients';

// 定义请求参数和返回类型
export interface MyData {
  id: string;
  name: string;
}

interface Params {
  page: number;
  size: number;
}

type CreateParams = Omit<MyData, 'id'>;
type UpdateParams = MyData;

// GET 请求
export const getMyData = v1Fetch.get<Params, MyData[]>('/my-data');

// 带路径参数的 URL（{var} 自动替换）
export const getDetail = v1Fetch.get<{ id: string }, MyData>('/my-data/{id}');

// POST 请求
export const createMyData = v1Fetch.post<CreateParams, MyData>('/my-data');

// PUT 请求
export const updateMyData = v1Fetch.put<UpdateParams, MyData>('/my-data/{id}');

// DELETE 请求
export const deleteMyData = v1Fetch.delete<{ id: string }, void>('/my-data/{id}');
```

### 3.3 在 Store 或页面中调用

```ts
// stores/my-feature.ts
import { defineStore } from 'pinia';
import { getMyData } from '~/api/modules/my-feature';

import type { MyData } from '~/api/modules/my-feature';

export const useMyStore = defineStore('my-feature', () => {
  const data = ref<MyData[]>([]);

  async function fetchData() {
    // 默认返回 res.data（拦截器已处理）
    data.value = await getMyData({ page: 1, size: 20 });
  }

  return { data, fetchData };
});
```

## 4. 请求配置 (Config)

通过第二个参数传入配置，覆盖默认值：

```ts
// 获取完整 response（含 code/status/data）
const res = await getMyData(params, { needRes: true });

// 获取原生 Response（流式/文件下载）
const res = await downloadFile(params, { originalResponse: true });

// 关闭自动错误提示，自行处理
const res = await getMyData(params, { interceptorErr: false });

// multipart 上传
const res = await uploadFile({ file }, { multipart: true });

// 返回 blob
const res = await exportData(params, { responseType: 'blob' });

// 校验业务码（res.code === 0 或 res.status === 0）
const res = await getMyData(params, { validateCode: true });

// 需要状态码信息
const res = await getMyData(params, { needStatus: true });

// GET 参数放 body
const res = await getMyData(bodyParams, { isBodyParam: true });

// 请求不可取消（不参与路由切换自动取消）
const res = await longPolling(params, { irrevocable: true });
```

### Config 扩展字段

| 字段               | 类型                   | 默认     | 说明                                           |
| ------------------ | ---------------------- | -------- | ---------------------------------------------- |
| `id`               | string                 | 自动生成 | 队列登记标识；当前不能作为请求合并或缓存键使用 |
| `interceptorErr`   | boolean                | true     | 是否自动弹 Message 错误提示                    |
| `irrevocable`      | boolean                | false    | 请求不可取消（跳过路由切换取消）               |
| `isBodyParam`      | boolean                | false    | GET/DELETE 参数放 body                         |
| `multipart`        | boolean                | false    | 非路径参数组装为 multipart/form-data           |
| `needRes`          | boolean                | false    | 返回完整 response（含 code/data/error）        |
| `needStatus`       | boolean                | false    | reject 时附带 status/statusText                |
| `originalResponse` | boolean                | false    | 返回原生 Response 对象                         |
| `prefix`           | string                 | -        | URL 前缀                                       |
| `responseType`     | 'json'\|'text'\|'blob' | 'json'   | 响应解析类型                                   |
| `validateCode`     | boolean                | false    | 校验业务码                                     |

`Config` 同时继承原生 `RequestInit`，因此还可传入 `headers`、`signal`、`credentials` 等浏览器 Fetch 配置。

## 5. URL 路径参数

URL 中的 `{var}` 格式参数自动从 params 中提取并替换：

```ts
// 定义
export const getApp = v1Fetch.get<{ appId: string; env: string }, App>('/apps/{appId}/envs/{env}');

// 调用 — appId 和 env 从 URL 替换，其余参数作为 query string
const app = await getApp({ appId: 'my-app', env: 'prod', detail: true });
// 实际请求: GET /apps/my-app/envs/prod?detail=true
```

## 6. 响应拦截器

`fetch.ts` 在模块加载时通过 `interceptors.response.use(...)` 注册统一响应处理；`interceptors.ts` 提供注册器和底层 Fetch 执行能力。

| HTTP 状态  | 处理                                                                        |
| ---------- | --------------------------------------------------------------------------- |
| 200        | 返回 `res.data`（默认）或 `res`（needRes）或 `Response`（originalResponse） |
| 401        | 跳转登录页 `BK_LOGIN_URL?c_url=当前URL`                                     |
| 403        | Message 错误提示「无权限」+ reject                                          |
| 400        | Message 错误详情（含 traceId）+ reject                                      |
| 其他非 2xx | Message 错误提示 + reject                                                   |

### 返回值处理逻辑

```ts
// 默认: 返回 res.data
const data = await getMyData(params);
// data 即为后端 data 字段内容

// needRes: 返回完整 response
const res = await getMyData(params, { needRes: true });
// res = { code, data, error, status, traceId }

// originalResponse: 返回原生 Response（流式/下载）
const response = await downloadFile(params, { originalResponse: true });
// 需自行 response.json() / response.blob()
```

### 错误处理

默认情况下，错误已由拦截器自动弹 Message 提示。如需自定义处理：

```ts
try {
  const data = await getMyData(params, { interceptorErr: false });
} catch (err) {
  // err 包含 res.error 信息和 traceId
  console.error(err.error?.message);
  // 自行展示错误 UI
}
```

## 7. 请求队列

`request-queue.ts` 用于跟踪和取消请求：

- **路由切换取消**：使用 `AbortController`，路由切换时自动取消队列中的未完成请求
- **不可取消请求**：设置 `irrevocable: true` 跳过取消（如轮询、关键操作）
- **id 仅用于队列登记**：相同 id 不会合并、复用或阻止第二个网络请求

> 当前自定义 `id` 还存在入队 id 与完成时移除 id 不一致的问题。修复实现前，不要依赖自定义 id 做去重或生命周期管理。

## 8. 链路追踪 (trace-id)

请求标识与响应链路标识由两个位置分别处理：

- `interceptors.ts` 为请求生成 `X-Bkapi-Request-Id`
- `trace-id.ts` 从响应头读取 `X-Trace-Id`
- `fetch.ts` 将响应 Trace ID 透传到 reject 对象，并附加到 Message 错误详情

不要把 `X-Bkapi-Request-Id` 与服务端返回的 `X-Trace-Id` 当作同一个字段。

## 9. 后端代理路径

| API 模块   | 前缀                                  | 说明                 |
| ---------- | ------------------------------------- | -------------------- |
| v1 API     | `{BK_API_PREFIX}/bkms/v1/bkms-server` | 核心 API（自动生成） |
| BCS API    | `/bcsapi`                             | 蓝鲸容器服务         |
| 制品库     | `/ms`                                 | BK Repo              |
| 用户选择器 | `/api-bk-user-selector`               | bk-user-web          |

## 10. 最佳实践

1. **优先使用自动生成** — v1 API 通过 `pnpm gen:api:v1` 生成，保持与后端同步
2. **类型先行** — 定义清晰的 Params 和 Return 类型
3. **URL 参数用 `{var}`** — 不要手动拼接 URL
4. **流式/下载用 originalResponse** — 避免拦截器预读 body
5. **上传用 multipart** — 自动处理 FormData
6. **关闭拦截器的场景** — 页面级错误展示、静默轮询
7. **关键操作设 irrevocable** — 避免路由切换导致请求中断
