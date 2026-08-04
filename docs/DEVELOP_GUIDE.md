## 服务治理开发环境搭建指南

本文档站在整体层面，为开发者搭建本地开发环境提供指引。如需查阅各服务更为细节的信息，请移步各服务子目录。

### 说明

参考本文档，开发者可自行搭建起一套（基本）完整的服务治理站点，正常访问前、后端功能，因此可以方便地体验产品功能，完成涉及多模块的联合调试。本地开发环境，有助于缩短反馈路径，提升研发效率。

开始搭建环境前，请准备：

- 一台 Linux 系统服务器（建议使用 DevCloud 机器）
- docker & docker-compose（或其他类似服务）
	- 用于启动 MongoDB、RabbitMQ 等数据库依赖
- Go 语言运行环境（推荐使用 [gvm](https://github.com/moovweb/gvm) 管理多版本）
- Node.js 运行环境（推荐 18 以上，建议使用 [nvm](https://github.com/nvm-sh/nvm) 管理版本）

#### 💡 重要提示

- 本文档着重从**原理和细节层面**搭建整套服务，因此并不追求速度（比如“一键/脚本搭建”），文中大多数步骤，需要手动执行命令或修改配置文件完成，这是有意为之
	- TODO：提供更简便的“测试环境搭建指南”，比如使用 Helm
- 本文档目前包含以下服务（更多服务待添加）：
  - bkms-server（trpc-go 服务，包含空间、环境、应用等基础模型的定义和管理，作为前端对接的唯一入口）
  - ui（基于 vue 前端）
- 推荐使用 [tmuxinator](https://github.com/tmuxinator/tmuxinator) 工具来启动每个服务
- 文档中所有涉及设置环境变量的部分，都使用 `export ...` 命令，你完全可以用其他方式（比如 .env 文件）来替代，只要能达到效果
- 公用的 AI coding agent 配置统一放置在仓库根目录的 `.agents` 目录下；如果本地使用的工具需要读取专属目录（如 `.codebuddy`、`.cursor`、`.claude` 等），请在本地创建指向 `.agents` 的软链，不要提交这些工具专属目录

### 依赖服务与准备工作

#### 使用 docker-compose 启动依赖服务

当前，服务治理站点依赖以下服务（为避免端口号冲突，不使用服务默认端口）：

- `MongoDB`：数据库，大多数服务依赖
	- *本文档使用端口：`27117`*
- `Redis`：缓存服务，部分服务依赖
    - *本文档使用端口：`26379`*
- `RabbitMQ`：队列服务，部分服务依赖
	- *本文档使用端口：`25672`*
- `etcd`：trpc-go 服务注册与发现插件依赖
	- *本文档使用端口：`29379`*

本文推荐使用 docker-compose 工具来启动和管理所有依赖服务，以下是一份示例配置文件：

```yaml
# 文件名：docker-compose.yaml
version: '3.2'
services:
  mongo_main:
    image: "mongo:8.0.9"
    volumes:
      - /home/bkmsusr/sysapps_store/mongo8:/data/db
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_ROOT_PASSWORD}
    ports:
      - "27117:27017"
  redis:
    image: "redis:7.4.5"
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "26379:6379"
  rabbitmq:
    image: "rabbitmq:3.13.7"
    volumes:
      - /home/bkmsusr/sysapps_store/rabbitmq:/var/lib/rabbitmq
    environment:
      RABBITMQ_DEFAULT_USER: admin
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD}
    ports:
      - "25672:5672" # AMQP port
      - "15672:15672" # Management UI port
  etcd:
    image: 'bitnami/etcd:3.5.15'
    environment:
      - ALLOW_NONE_AUTHENTICATION=yes
      - ETCD_ADVERTISE_CLIENT_URLS=http://etcd:2379
    ports:
      - "29379:2379"
      - "29380:2380"
```

基于这份配置，建议修改以下内容：

- `/home/bkmsusr/sysapps_store/` 目录，用于在主机上持久化存放数据文件，建议调整为你自己的目录
- `${MONGO_ROOT_PASSWORD}`，表示该值将通过同目录中的 `.env` 文件，从环境变量中读取，建议新建 `.env` 变量文件
	- **文档之后任何出现数据库密码的地方，将会以 {PASS} 替代**

变量文件 `.env` 示例：

```
MONGO_ROOT_PASSWORD={某个随机生成的密码}
REDIS_PASSWORD={某个随机生成的密码}
RABBITMQ_PASSWORD={某个随机生成的密码}
```

准备好配置文件后，执行 `docker-compose up -d` 来启动所有依赖服务。

#### hosts 配置与服务端口说明

为了方便调试，服务治理站点的前后端需共享使用一个统一的域名，为便于身份认证，**该域名必须使用 `.bkms.example.com` 根域**，比如 `local-dev.bkms.example.com`。

在用于**部署和访问**服务治理站点的机器上，修改 `/etc/hosts` 文件，添加以下条目：

```
# 修改为真实的用于搭建服务治理的 IP
xx.xx.xx.xx   local-dev.bkms.example.com
```

如果完全依据本文档来搭建整个项目，将占用以下端口：

- MongoDB：27117
- Redis：26379 
- RabbitMQ：25672
- etcd：29379
- bkms-server: 31302
- ui: 5173

### 各服务启动指引

#### bkms-server

进入项目的 `apps/bkms-server` 目录，设置以下环境变量：

```bash
# 服务发现 etcd 地址
export NAMING_ETCD_ADDRESS="127.0.0.1:29379"
# 服务所启动的默认端口
export BKMS_BKMS_SERVER_REST_PORT=31302

# 用于执行单元测试的测试数据库地址，部分测试用例强依赖
export FOR_TEST_DB_URL="mongodb://root:pass@localhost:27117"
```

参考 configs 目录下的 config.yaml 文件，创建 `config.local.yaml`。示例内容如下：

```yaml
# 蓝鲸应用配置（可从开发者中心获取）
bkApp:
  # 应用 Code
  code: bkms
  # 应用 Secret
  secret: masked
# 蓝鲸平台地址配置
bkPlatUrls:
  # 蓝鲸 API 地址模板
  bkApiUrlTmpl: http://{api_name}.apigw.example.com
# 蓝鲸 API 环境配置
bkApiStages:
  # BCS Cluster Resources
  clusterresources: prod
  # 其他的按需要加，可参考 configs/config.yaml ...
# 加密配置
encrypt:
  # 加密密钥（base64 编码后的 32 个随机字节）
  # 可以通过执行以下 Python 命令生成：
  #
  #     python -c "import base64, random;print(base64.b64encode(random.randbytes(32)).decode())"
  #
  # 或者也可以直接使用下面的固定值（不推荐，仅限本地测试环境使用）
  secret: 15RQgSBLUfCBjhcGzzFQu5WScClbSVrVzoEkWbJJhB4=
# 数据库相关配置
mongo:
  serviceName: trpc.bkms.mongodb.bkmsserver
  username: root
  password: {PASS}
  host: 127.0.0.1
  port: 27117
  database: bkms
# Redis 相关配置，主要用于数据缓存
redis:
  host: 127.0.0.1
  port: 26379
  db: 0
  password: {PASS}
  # 超时配置推荐使用默认值，如有需要也可自行配置
  # dialTimeout: 5
  # readTimeout: 2
  # writeTimeout: 2
# RabbitMQ 相关配置，主要用于消息队列
rabbitmq:
  host: 127.0.0.1
  port: 25672
  username: admin
  password: {PASS}
  # 推荐使用具体的 vhost 如：bkms-server
  # 需要手动登录 rabbitmq 管理界面创建
  # 本地开发也可以直接使用 / 这个 vhost
  vhost: /
# 轮询相关配置
taskPoller:
  # 部署状态
  deployStatus:
    timeout: 1200
    interval: 15
development:
  useStubPerm: true
  useKubeConfigCluster: true
```

- 修改其中的 mongo 部分配置，调整为可以正常连接的 mongo 服务器配置
- 当 useStubPerm 配置项被启用后，服务将默认对所有权限校验放行（比如总是允许任何人修改任何应用）
- 当 useKubeConfigCluster 配置项被启用后，服务将使用 kubeconfig 文件中指定的集群，其默认路径为 `~/.kube/config`，如需特殊指定则可配置 `development.stubKubeConfigPath`（如：`/tmp/kubeconfig`）

一切就绪后，执行以下命令启动服务：

```shell
# 构建 proto
make proto
# 构建二进制可执行文件
make build
# 启动 web 服务器
./build/bkms-server webserver --srvCfg ./configs/config.local.yaml
# 启动后台异步任务 worker
./build/bkms-server worker --srvCfg ./configs/config.local.yaml
```

打开浏览器，访问 `http://local-dev.bkms.example.com:31302/` 来确认服务成功启动。

#### UI（前端）

进入项目的 `apps/ui` 目录，参考其中的 README 文件安装 pnpm 命令行工具，之后执行 `pnpm install` 以安装依赖包。

然后，设置以下环境变量：

```bash
# 服务所使用的域名，建议使用 bkms.example.com 根域
export BK_ALLOWED_HOST=local-dev.bkms.example.com
# 服务所监听的主机地址，修改后监听所有网络地址
export BK_APP_HOST=0.0.0.0
# 服务所监听的端口
export BK_APP_PORT=5173
# 其所依赖的后端服务访问地址，直接指向 bkms-server
export BK_API_BASE_URL="http://local-dev.bkms.example.com:31302"

# 其他变量
export BK_NODE_ENV="development"
export BK_BCS="https://bcs.example.com"
export BK_REPO_URL="https://bkrepo.example.com"
export BK_DEVOPS="https://devops.example.com"
```

> 更多详情，详见 `vite.config.mts` 配置文件。

最后，执行 `pnpm dev` 命令以启动开发服务器。如果一切正常，访问 `http://local-dev.bkms.example.com:5173` 来打开服务治理首页。
