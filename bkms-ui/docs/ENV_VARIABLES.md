# bkms-ui 环境变量说明

## 1. 变量分类

项目环境变量分为两类：

| 类型           | 注入时机                           | 可运行时覆盖 | 格式                            |
| -------------- | ---------------------------------- | ------------ | ------------------------------- |
| **构建时变量** | `docker build` / `pnpm build`      | 否           | 直接写入产物                    |
| **运行时变量** | 容器启动（`docker-entrypoint.sh`） | 是           | `__BKMS_RT_BK_XXX__` 占位符替换 |

## 2. 构建时变量

构建时确定，不可在已生成的静态产物中覆盖。

| 变量                            | 说明                                                | 默认  |
| ------------------------------- | --------------------------------------------------- | ----- |
| `BK_NODE_ENV`                   | 固定 `production`（生产构建）                       | -     |
| `BK_STATIC_URL`                 | Vite `base` 路径，Dockerfile 强制为空（根路径部署） | `''`  |
| `BKMS_APP_VERSION`              | 构建时注入版本号（`--build-arg`）                   | `--`  |
| `BK_CI_GIT_REPO_HEAD_COMMIT_ID` | Git commit hash（`--build-arg`，可选）              | `--`  |
| `BK_CI_BUILD_NUM`               | 构建号（`--build-arg`，可选）                       | `dev` |

以下变量只参与 `vite.config.mts` 的开发服务器配置，也不是生产静态站点的运行时配置：

| 变量              | 用途                                    |
| ----------------- | --------------------------------------- |
| `BK_API_BASE_URL` | `/bkms`、`/simple_account` 开发代理目标 |
| `BK_APP_HOST`     | Vite dev server 监听地址                |
| `BK_APP_PORT`     | Vite dev server 端口                    |
| `BK_ALLOWED_HOST` | Vite dev server 允许访问的 Host         |

### Vite 环境变量前缀

所有 `BK_` 前缀的变量通过 Vite 注入 `import.meta.env`：

```ts
// vite.config.mts
envPrefix: 'BK_';
```

```ts
// 代码中使用
const apiPrefix = import.meta.env.BK_API_PREFIX;
```

## 3. 生产运行时变量

只有同时满足以下条件的变量，才能在容器启动时覆盖：

1. `.env.production` 中使用 `__BKMS_RT_BK_XXX__` 占位符
2. 变量被打包进最终 HTML / JavaScript / CSS 产物
3. `docker-entrypoint.sh` 能识别该占位符名称

容器启动时，`docker-entrypoint.sh` 会在静态产物中查找占位符并替换为实际值。

### 3.1 完整清单

| 变量                        | 说明                               | 使用位置                     |
| --------------------------- | ---------------------------------- | ---------------------------- |
| `BK_API_PREFIX`             | API 前缀                           | `api/clients.ts`             |
| `BK_SITE_URL`               | 网站 route 前缀                    | `modules/router.ts`          |
| `BK_SHARED_RES_BASE_JS_URL` | 平台配置 JS 地址（项目名称/title） | `index.html`                 |
| `BK_DEVOPS`                 | 蓝盾项目地址                       | `index.html`                 |
| `BK_BCS`                    | 容器项目地址                       | `index.html`                 |
| `BK_MONITOR`                | 监控项目地址                       | `index.html`                 |
| `BK_BCS_API_BASE_URL`       | BCS API 地址                       | `vite.config.mts` proxy      |
| `BK_LOGIN_URL`              | 登录地址（401 跳转 + 退出登录）    | `api/fetch.ts`、`index.html` |
| `BK_POLARIS_URL`            | 北极星项目地址                     | `index.html`                 |
| `BK_DOC_URL`                | 文档地址                           | `index.html`                 |
| `BK_API_URL_TMPL`           | 蓝鲸 API 网关模板 `{api_name}`     | `vite.config.mts` proxy      |
| `BK_REPO_URL`               | 代码库数据源地址                   | `vite.config.mts` proxy      |
| `BK_GOLANG_PROXY_URL`       | Go 代理地址                        | 运行时                       |

### 3.2 当前限制

| 变量               | 当前状态                                                                                         | 建议                                             |
| ------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------ |
| `BK_API_BASE_URL`  | 只用于开发代理；即使 `.env.production` 中存在占位符，也不会进入生产静态产物                      | 从生产运行时配置中移除，保留为开发变量           |
| `BK_API_V1_PREFIX` | `clients.ts` 支持读取，但 `.env.production` 没有占位符；名称含数字，也不匹配 entrypoint 当前正则 | 需要运行时覆盖时，同时补充占位符并让正则支持数字 |
| `BK_BSCP_URL`      | 页面代码会读取，但生产配置固定为空，不能运行时覆盖                                               | 改为运行时占位符，或明确由构建阶段注入           |

### 3.3 index.html 中的运行时变量

`index.html` 通过 `<script>` 标签暴露全局变量，构建时为 `%BK_XXX%` 占位符，生产环境由 entrypoint 替换：

```html
<script>
  var BK_LOGIN_URL = '%BK_LOGIN_URL%';
  var BK_API_PREFIX = '%BK_API_PREFIX%';
  var BK_SHARED_RES_BASE_JS_URL = '%BK_SHARED_RES_BASE_JS_URL%';
  var BK_DEVOPS = '%BK_DEVOPS%';
  var BK_BCS = '%BK_BCS%';
  var BK_MONITOR = '%BK_MONITOR%';
  var BK_POLARIS_URL = '%BK_POLARIS_URL%';
  var BK_DOC_URL = '%BK_DOC_URL%';
  var BK_BKMS_APP_VERSION = '__BK_BKMS_APP_VERSION__';
</script>
```

> 代码中通过 `window.BK_LOGIN_URL` 等方式访问。

### 3.4 添加新的运行时变量

1. 在 `.env.production` 中添加占位符：

   ```env
   BK_NEW_VAR = '__BKMS_RT_BK_NEW_VAR__'
   ```

2. 在 `src/vite-env.d.ts` 中注册类型（如需在 `import.meta.env` 中使用）

3. 确认变量确实在应用代码或 `index.html` 中使用，否则占位符不会进入构建产物

4. 检查 `docker-entrypoint.sh` 的占位符正则。当前仅匹配 `[A-Z_]*`，变量名包含数字时必须先扩展为支持 `[A-Z0-9_]*`

## 4. 本地开发环境

仓库提供 `.env.development` 作为共享开发配置。个人地址和覆盖项建议放在不提交的 `.env.development.local` 中，Vite 会在 development mode 自动加载并覆盖同名变量：

```env
# .env.development.local 示例
BK_API_PREFIX = 'http://localhost:8080'
BK_API_BASE_URL = 'http://localhost:8080'
BK_LOGIN_URL = 'http://login.example.com'
BK_DEVOPS = 'https://devops.example.com'
BK_BCS = 'https://bcs.example.com'
BK_MONITOR = 'https://monitor.example.com'
BK_BCS_API_BASE_URL = 'https://bcs-api.example.com'
BK_POLARIS_URL = 'https://polaris.example.com'
BK_DOC_URL = 'https://docs.example.com'
BK_API_URL_TMPL = 'https://{api_name}.apigw.example.com'
BK_REPO_URL = 'https://devops.example.com'
BK_GOLANG_PROXY_URL = 'https://goproxy.example.com'
BK_APP_HOST = '0.0.0.0'
BK_APP_PORT = '5008'
BK_ALLOWED_HOST = 'localhost'
```

### Vite Dev Server 配置

| 变量              | 说明                                   |
| ----------------- | -------------------------------------- |
| `BK_APP_HOST`     | dev server 监听地址                    |
| `BK_APP_PORT`     | dev server 端口（仓库开发配置为 5008） |
| `BK_ALLOWED_HOST` | `server.allowedHosts`                  |

## 5. 生产部署

### 5.1 Docker 构建

```bash
docker build -f Dockerfile.prod \
  --platform linux/amd64 \
  --build-arg BKMS_APP_VERSION=1.0.0 \
  --build-arg BK_CI_GIT_REPO_HEAD_COMMIT_ID="$(git rev-parse HEAD)" \
  --build-arg BK_CI_BUILD_NUM=local \
  -t bkms-ui:<tag> .
```

### 5.2 运行时注入（三选一）

**方式一：挂载 .env 文件（推荐）**

```bash
docker run --rm -p 5000:5000 \
  -v /path/to/.env.staging:/env/.env:ro \
  bkms-ui:<tag>
```

`.env.staging` 格式（支持空格和引号）：

```env
BK_API_PREFIX = 'https://stag.bkms.example.com'
BK_LOGIN_URL = 'https://login.example.com'
```

**方式二：单个环境变量**

```bash
docker run --rm -p 5000:5000 \
  -e BK_API_PREFIX=https://api.example.com \
  -e BK_LOGIN_URL=https://login.example.com \
  bkms-ui:<tag>
```

**方式三：Docker --env-file（KEY=VALUE 格式）**

```bash
docker run --rm -p 5000:5000 \
  --env-file .env.docker \
  bkms-ui:<tag>
```

### 5.3 优先级

**单个 `-e` > `.env` 文件**

同时使用时，`-e` 设置的同名变量不会被 `.env` 文件覆盖。

### 5.4 Kubernetes 部署

```yaml
spec:
  containers:
    - name: bkms-ui
      image: bkms-ui:<tag>
      ports:
        - containerPort: 5000
      envFrom:
        - configMapRef:
            name: bkms-ui-env
```

## 6. 运行时注入原理

```
构建阶段:
  .env.production 中 BK_NEW_VAR = '__BKMS_RT_BK_NEW_VAR__'
  → Vite 构建产物中保留占位符 __BKMS_RT_BK_NEW_VAR__

运行阶段:
  docker-entrypoint.sh
  1. 读取 /env/.env (或 ENV_FILE 指定路径)
  2. 自动扫描产物中 __BKMS_RT_* 占位符
  3. 用实际环境变量值替换
  4. 启动 Nginx
```

## 7. 验证运行时注入

```bash
# 启动容器并注入变量
docker run --rm -d -p 5000:5000 \
  -e BK_LOGIN_URL=https://login.test.com \
  --name bkms-ui-test bkms-ui:<tag>

# 检查 index.html 中的值是否被替换
docker exec bkms-ui-test grep 'BK_LOGIN_URL' /usr/share/nginx/html/index.html
# 期望输出: var BK_LOGIN_URL = 'https://login.test.com'

docker stop bkms-ui-test
```

## 8. 常见问题

| 现象                                 | 原因                                                                                |
| ------------------------------------ | ----------------------------------------------------------------------------------- |
| 白屏或接口前缀错误                   | 运行时未注入 `BK_API_PREFIX` 等变量                                                 |
| 页面显示 `__BKMS_RT_BK_XXX__` 占位符 | 容器启动时未执行 entrypoint（如 `--entrypoint` 覆盖）                               |
| `.env` 文件变量未生效                | 检查 `ENV_FILE` 路径及变量名；脚本同时支持 `KEY=VALUE` 和项目常用的 `KEY = 'value'` |
| 静态资源 404                         | `BK_STATIC_URL` 配置问题；生产镜像固定为根路径部署                                  |
| dev server 无法访问后端              | 检查 `.env.development` 中 proxy 目标地址                                           |

## 9. 相关文档

- [Docker 生产部署完整说明](../DEPLOY.md)
- [开发指南](./DEVELOPMENT.md)
- [API 指南](./API_GUIDE.md)
