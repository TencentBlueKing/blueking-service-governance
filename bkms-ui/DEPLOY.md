# BKMS UI 生产部署说明

本文说明如何通过 **Docker 多阶段构建** 将本仓库前端以 **Nginx 静态站点** 形式部署到生产环境。核心特性：**一次构建、多环境部署** —— 运行时通过环境变量或 `.env` 文件注入配置，无需为每个环境单独构建镜像。

相关文件：

| 文件 | 说明 |
|------|------|
| `Dockerfile.prod` | 生产镜像：Node 构建 + `nginx:1.30-alpine` 运行 |
| `docker/nginx-default.conf` | 容器内 Nginx 站点配置（端口、缓存、安全头） |
| `docker/docker-entrypoint.sh` | 容器启动入口：注入运行时环境变量 |
| `.env.production` | 构建时环境占位符定义 |

---

## 1. 架构简述

- **构建阶段**：`node:23-alpine` 中执行 `pnpm install --frozen-lockfile` 与 `pnpm build --mode production`，产出 `dist/`。运行时变量在构建产物中保留为 `__BKMS_RT_BK_XXX__` 占位符。
- **运行阶段**：`docker-entrypoint.sh` 扫描构建产物中的占位符，用实际环境变量替换后启动 Nginx。无 Node 运行时。
- **监听端口**：**5000**（与历史 `targetPort` / `http-server` 一致，便于沿用编排配置）。
- **健康检查**：镜像内置 `HEALTHCHECK`，对 `http://127.0.0.1:5000/` 发起探测；编排层可映射为存活/就绪探针。

---

## 2. 环境变量设计

### 2.1 构建时变量（镜像构建时确定，不可运行时覆盖）

| 变量 | 说明 |
|------|------|
| `BK_NODE_ENV` | 固定为 `production`，用于代码条件分支 |
| `BK_STATIC_URL` | Vite `base` 路径，Dockerfile 强制为空（根路径部署） |
| `BKMS_APP_VERSION` | 构建时注入版本号（`--build-arg`） |
| `BK_CI_GIT_REPO_HEAD_COMMIT_ID` | Git commit（`--build-arg`，可选） |
| `BK_CI_BUILD_NUM` | 构建号（`--build-arg`，可选） |

### 2.2 运行时变量（容器启动时注入）

以下变量在 `.env.production` 中以 `__BKMS_RT_BK_XXX__` 占位符形式写入构建产物，容器启动时由 `docker-entrypoint.sh` 替换为实际值。未提供的变量自动清空为空字符串。

| 变量 | 说明 |
|------|------|
| `BK_API_PREFIX` | API 前缀 |
| `BK_API_BASE_URL` | BKMS API 服务地址（CLI apiserver 等） |
| `BK_SITE_URL` | 网站 route 前缀 |
| `BK_SHARED_RES_BASE_JS_URL` | 平台配置地址（项目名称和 title） |
| `BK_DEVOPS` | 蓝盾项目地址 |
| `BK_BCS` | 容器项目地址 |
| `BK_MONITOR` | 监控项目地址 |
| `BK_BCS_API_BASE_URL` | BCS API 地址 |
| `BK_LOGIN_URL` | 登录地址（关联退出登录） |
| `BK_POLARIS_URL` | 北极星项目地址 |
| `BK_DOC_URL` | 文档地址 |
| `BK_IAM_URL` | 权限中心地址 |
| `BK_API_URL_TMPL` | 蓝鲸 API 网关模板 |
| `BK_REPO_URL` | 代码库数据源地址 |
| `BK_GOLANG_PROXY_URL` | Go 代理地址 |

### 2.3 添加新的运行时变量

1. 在 `.env.production` 中添加 `BK_NEW_VAR = '__BKMS_RT_BK_NEW_VAR__'`
2. 在 `src/vite-env.d.ts` 中注册类型
3. 无需修改 `docker-entrypoint.sh` —— 脚本自动发现 `__BKMS_RT_*` 占位符

---

## 3. 构建镜像

在项目根目录（`apps/ui`）执行。

### 3.1 目标架构（Linux x86_64 服务器）

常见生产 **Linux 主机为 x86_64**，容器平台对应 **`linux/amd64`**。

若在 **Apple Silicon（M 系列）Mac** 或 **arm64 Linux** 上直接执行 `docker build` 且**不**指定 `--platform`，默认产出 **`linux/arm64`** 镜像。推送到镜像仓库后，在 **x86_64 服务器** 会出现：

`no matching manifest for linux/amd64 in the manifest list entries`

**面向主流 Linux 服务器发布时，构建命令中应包含 `--platform linux/amd64`**。

### 3.2 构建命令

```bash
docker build -f Dockerfile.prod \
  --platform linux/amd64 \
  --build-arg BKMS_APP_VERSION=1.0.0 \
  --build-arg BK_CI_GIT_REPO_HEAD_COMMIT_ID="$(git rev-parse HEAD 2>/dev/null || echo unknown)" \
  --build-arg BK_CI_BUILD_NUM=local \
  -t bkms-ui:<tag> .
```

> 注意：**不再需要 `--build-arg MODE=xxx`**。生产镜像统一使用 `production` 模式构建，环境差异通过运行时注入。

---

## 4. 运行容器（运行时环境注入）

### 4.1 方式一：挂载 `.env` 文件（推荐）

将项目格式的 `.env` 文件挂载到容器默认路径 `/env/.env`，无需额外参数：

```bash
docker run --rm -p 5000:5000 \
  -v /path/to/.env.staging:/env/.env:ro \
  bkms-ui:<tag>
```

如需自定义文件路径，通过 `ENV_FILE` 指定：

```bash
docker run --rm -p 5000:5000 \
  -e ENV_FILE=/custom/path/.env \
  -v /path/to/.env.staging:/custom/path/.env:ro \
  bkms-ui:<tag>
```

`.env.staging` 示例：

```env
BK_API_PREFIX = 'https://stag.bkms.example.com'
BK_LOGIN_URL = 'https://login.example.com'
BK_DEVOPS = 'https://devops.example.com'
BK_BCS = 'https://bcs.example.com'
BK_MONITOR = 'https://monitor.example.com'
BK_DOC_URL = 'https://docs.example.com'
BK_BCS_API_BASE_URL = 'https://bcs-api.example.com'
BK_POLARIS_URL = 'https://polaris.example.com'
BK_API_URL_TMPL = 'https://{api_name}.apigw.example.com'
BK_REPO_URL = 'https://devops.example.com'
BK_GOLANG_PROXY_URL = 'https://goproxy.example.com'
```

### 4.2 方式二：单个指定环境变量

```bash
docker run --rm -p 5000:5000 \
  -e BK_API_PREFIX=https://api.example.com \
  -e BK_LOGIN_URL=https://login.example.com \
  -e BK_DEVOPS=https://devops.example.com \
  bkms-ui:<tag>
```

### 4.3 方式三：Docker `--env-file`（KEY=VALUE 格式）

```bash
docker run --rm -p 5000:5000 \
  --env-file .env.docker \
  bkms-ui:<tag>
```

> **注意**：Docker 的 `--env-file` 要求 `KEY=VALUE` 格式（无空格、无引号），与项目 `.env` 格式不同。

`.env.docker` 示例：

```env
BK_API_PREFIX=https://api.example.com
BK_LOGIN_URL=https://login.example.com
```

### 4.4 组合使用与优先级

可同时挂载 `.env` 文件和单个 `-e`，**单个 `-e` 优先级高于 `.env` 文件**：

```bash
docker run --rm -p 5000:5000 \
  -v ./env.staging:/env/.env:ro \
  -e BK_API_PREFIX=https://override.example.com \
  bkms-ui:<tag>
```

上例中 `BK_API_PREFIX` 使用 `-e` 的值，其余变量从 `.env.staging` 读取。

### 4.5 Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: bkms-ui
          image: bkms-ui:<tag>
          ports:
            - containerPort: 5000
          envFrom:
            - configMapRef:
                name: bkms-ui-env
          # 或挂载 .env 文件到默认路径 /env/.env
          # volumeMounts:
          #   - name: env-config
          #     mountPath: /env/.env
          #     subPath: .env
          #     readOnly: true
```

---

## 5. 本地容器运行时（Colima）

在 **macOS** 上若以 **Colima** 作为 Docker 引擎（替代 Docker Desktop），`docker build` / `docker run` 均通过 Colima 提供的 API 执行。

1. **启动**（未运行时先执行）  
   `colima start`  
   可按机器情况增加资源，例如：`colima start --cpu 4 --memory 8 --disk 60`。

2. **确认上下文**  
   `docker context ls`  
   当前为 `colima` 时，客户端会使用 Colima 的套接字。

3. **常见故障**  
   - 报错无法连接 Docker API：先 `colima status`，若未运行则 `colima start`。  
   - 升级或异常后：可尝试 `colima stop && colima start`。

4. **跨平台构建（`--platform linux/amd64`）**  
   在 **Apple Silicon** 上为 Linux x86 服务器打镜像时，Colima 会通过 QEMU 模拟 amd64，**首次构建可能较慢**。

---

## 6. Nginx 行为摘要（生产容器内）

配置见 `docker/nginx-default.conf`，要点如下：

- **SPA**：`try_files $uri /index.html`，前端路由由 `index.html` 承载。
- **缓存**：仅 **`/assets/`** 下构建产物长期缓存并带 `immutable`；**HTML 与根路径回退**使用 `no-cache`，避免用户长期命中旧入口导致资源不一致。
- **安全**：拒绝 **`.map`**、拒绝除 **`.well-known`** 外的 **隐藏路径**；对静态与 HTML 相关 `location` 补齐 **nosniff、frame、Referrer-Policy、Permissions-Policy** 等（**未内置 CSP/HSTS**：CSP 需按业务与外链单独设计；**HSTS 建议在 TLS 终止层** 由网关或 CDN 下发）。

若上层网关 / Ingress **已下发相同或更严的安全头**，注意避免重复或冲突。

---

## 7. 本地构建与验证

以下均在项目根目录 **`apps/ui`** 下执行。将示例中的 `<tag>` 换成你本地镜像标签（如 `local`）。

### 7.1 构建 → 注入环境 → 运行

```bash
# 1）构建（只需一次）
docker build -f Dockerfile.prod \
  --platform linux/amd64 \
  --build-arg BKMS_APP_VERSION=1.0.0 \
  -t bkms-ui:<tag> .

# 2）使用 .env.test 运行（测试环境配置）
docker run --rm -p 5000:5000 \
  -v "$PWD/.env.test:/env/.env:ro" \
  --name bkms-ui-local bkms-ui:<tag>

# 3）使用 .env.staging 运行（预发布环境）—— 同一镜像，不同配置
docker run --rm -p 5000:5000 \
  -v "/path/to/.env.staging:/env/.env:ro" \
  --name bkms-ui-local bkms-ui:<tag>
```

浏览器访问 **<http://127.0.0.1:5000/**，确认首屏与前端路由。>

### 7.2 校验 Nginx 配置语法

```bash
docker run --rm bkms-ui:<tag> nginx -t
```

> 注意：使用 ENTRYPOINT 后，需要覆盖入口才能直接执行 nginx 命令：
> `docker run --rm --entrypoint nginx bkms-ui:<tag> -t`

### 7.3 用 curl 检查响应头

容器运行时执行：

```bash
curl -sI http://127.0.0.1:5000/
curl -sI http://127.0.0.1:5000/index.html
curl -sI http://127.0.0.1:5000/__spa_route_probe__/not-a-real-file
```

| 请求 | 典型预期 |
|------|----------|
| `/`、`/index.html`、SPA 路径 | `Cache-Control: no-cache`；含安全头 |
| `/assets/*.{js,css,...}` | `Cache-Control` 含 `public`、长 max-age、`immutable` |

### 7.4 安全相关 spot check

```bash
curl -sI http://127.0.0.1:5000/.git/config  # 期望 403/404
```

### 7.5 镜像 HEALTHCHECK

```bash
docker inspect --format '{{json .State.Health}}' bkms-ui-local
```

### 7.6 只改 Nginx、少重建业务镜像

```bash
pnpm install && pnpm build --mode production

docker run --rm -p 5000:5000 \
  -v "$PWD/dist:/usr/share/nginx/html:ro" \
  -v "$PWD/docker/nginx-default.conf:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.30-alpine
```

### 7.7 验证运行时注入是否生效

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

---

## 8. 常见问题

| 现象 | 可能原因 |
|------|----------|
| 拉取镜像报错 **`no matching manifest for linux/amd64`** | 镜像仅为 arm64。重新用 `--platform linux/amd64` 构建。 |
| 白屏或接口前缀错误 | 运行时未注入 `BK_API_PREFIX` 等变量。检查 `-e` 或 `ENV_FILE` 是否正确。 |
| 页面显示 `__BKMS_RT_BK_XXX__` 占位符 | 容器启动时未执行 entrypoint（如使用了 `--entrypoint` 覆盖）。 |
| `.env` 文件变量未生效 | 检查文件格式是否正确（`KEY = 'value'`），以及 `ENV_FILE` 路径是否存在。 |
| 单个 `-e` 被 ENV_FILE 覆盖 | 不会发生。`-e` 优先级高于 ENV_FILE。 |
| 静态资源 404 | `base` 路径问题；生产镜像固定为根路径部署（`BK_STATIC_URL=''`）。 |
| 健康检查失败 | 平台探针端口未指向 5000。 |
| 构建失败 `frozen-lockfile` | 本地改过依赖未更新 `pnpm-lock.yaml`。 |
| `Cannot connect to the Docker daemon` | Colima 未启动：`colima start`。 |

---

## 9. 版本与维护

- **Nginx 基础镜像**：`nginx:1.30-alpine`（stable 线）。
- **运行时变量**：新增变量只需在 `.env.production` 中添加占位符，`docker-entrypoint.sh` 自动发现。
- **策略变更**：同步更新 `docker/nginx-default.conf`。

如有环境与配置中心规范，以**平台负责人**要求为准。
