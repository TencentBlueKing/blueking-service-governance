## API 测试

bkms-server 项目的 API 测试由 [bruno](https://www.usebruno.com/) 工具完成。bruno 是一个类似于 Postman 的 RESTful API 请求工具。它主要拥有以下特性：

- 支持同时使用功能 GUI 客户端和 CLI 工具两种方式
- 支持使用 JavaScript 语言编写复杂逻辑，实现灵活的请求配置
- 支持使用 assert 和 Chai 语法来编写断言语句，测试 API 的正确性

### 快速开始

#### 使用 CLI 执行测试

首先，参考[官方文档](https://docs.usebruno.com/bru-cli/installation)安装 bru 命令行工具。然后，进入 apis 目录，创建一份和你的测试环境相关的环境配置文件。

建议可以基于 `environments/local.bru` 克隆一份，对其中配置项进行微调：

```bru
vars {
  serviceURL: {...修改为可访问的 bkms-server API 服务地址...}
  workspace: api-test-ws
  appName: api-test-app
}
```

环境文件准备好以后，在 apis 目录执行 `bru run --env-file ./environments/{YOUR_ENV_FILE}.bru`。该命令将运行当前 collection 中的所有 API 测试，打印测试结果。

#### 启用真实部署类 API 测试

部分 API 测试会真实驱动部署流程，依赖可用的 Kubernetes 集群并且耗时较长。默认执行 `bru run` 时，这些目录会被跳过；如需启用，可在执行命令前设置对应环境变量：

```bash
# 启用 Helm 部署生命周期测试
RUN_HELM_DEPLOY_API_TESTS=1 bru run --env-file ./environments/{YOUR_ENV_FILE}.bru

# 启用 Trpc / TAF AppModel 部署生命周期测试
RUN_TRPC_DEPLOY_API_TESTS=1 RUN_TAF_DEPLOY_API_TESTS=1 bru run --env-file ./environments/{YOUR_ENV_FILE}.bru

# 启用全部真实部署类 API 测试
RUN_HELM_DEPLOY_API_TESTS=1 RUN_TRPC_DEPLOY_API_TESTS=1 RUN_TAF_DEPLOY_API_TESTS=1 \
  bru run --env-file ./environments/{YOUR_ENV_FILE}.bru
```

未设置这些变量时，仅跳过对应的真实部署生命周期目录；不实际部署工作负载的失败分支、参数校验类用例仍会正常执行。

如果要运行 Trpc / TAF AppModel 部署生命周期测试，需要在目标集群中预先安装 hook-operator 与 gamedeployment 组件：

```bash
# 注：推荐使用 bcs 在 Helm Chart 公共仓库中提供的最新版本
helm install bcs-gamedeployment-operator -n bcs-system bcs-gamedeployment-operator-1.31.0-alpha.90.tgz
helm install bcs-hook-operator -n bcs-system bcs-hook-operator-1.31.0-alpha.27.tgz
```

Helm / Trpc / TAF 部署生命周期测试都需要拉取工作负载镜像。启用 `RUN_HELM_DEPLOY_API_TESTS`、`RUN_TRPC_DEPLOY_API_TESTS` 或 `RUN_TAF_DEPLOY_API_TESTS` 中任一变量时，必须通过 `BKMS_APITEST_WORKLOAD_IMAGE` 提供一个目标集群节点可拉取的镜像仓库地址，**格式为不带 tag 的 `<registry>/<name>`**，例如 `example.com/bkms/apitest-workload`。

要求：

- 该 registry 中必须**预先存在 `1.0.0` 与 `2.0.0` 两个 tag** 的镜像（v1 用例使用 `1.0.0`，v2 / rollback 用例使用 `2.0.0`，tag 名称在测试代码中固定，不可通过环境变量覆盖）。
- 两个 tag 对应的镜像都必须包含 `sleep` 命令，测试会以 `sleep 3600` 作为容器启动命令，避免业务容器立即退出。

> Tip：为避免 API 测试数据与你的开发环境发生冲突，建议专门为 API 测试启动一个独立的 bkms-server 服务实例，连接不同数据库。

### 一键拉起 E2E 测试

基于 docker-compose 实现的 E2E 测试，可以在安装了 docker 的环境中一键运行所有 API 测试，无需手动启动任何服务。

E2E 栈完全自包含：bkms-server / worker、db-migrate、migration、bruno runner、mongo / redis、chartmuseum 全部跑在独立的 compose project `bkms-apitest` 与独立网络 `bkms-apitest-net` 中，bkms-server API 宿主端口为 `32402`，与单元测试依赖栈（`tests/utdeps/`、`tests/utdeps_db/`）完全隔离。

#### 前置条件

1. 宿主机已启动一个 K8s 集群（kind / k3d / docker-desktop / 远端集群均可），apiserver 监听在 docker bridge 可达的网络接口（例如 `0.0.0.0:<port>`，而不是仅 `127.0.0.1`）。容器内通过 `kubernetes.default → host-gateway` 触达它。
2. 宿主机有 kubeconfig（路径默认 `${HOME}/.kube/config`），其中 `server` 字段写为 `https://kubernetes.default:<port>`。
3. 已安装 `docker` / `docker-compose`、[`just`](https://github.com/casey/just)。
4. 已构建本地待测镜像（如 `bkms-server:my-dev-tag`）。

#### 启动 / 关闭

```bash
cd tests/

# 设置待测镜像（也可通过 .env 文件持久化）
export BKMS_SERVER_IMAGE=bkms-server:my-dev-tag

# 可选：覆盖默认 kubeconfig 路径
export BKMS_APITEST_KUBECONFIG=${HOME}/.kube/config

# 启动
just up

# 可选：使用本地二进制启动，避免每次本地改动后重新构建 bkms-server 镜像。
# 默认读取 ../build/bkms-server，可在仓库根目录执行 make build-dev 生成。
# 如果二进制在其他位置，可以通过 BKMS_LOCAL_BINARY 覆盖。
# (cd .. && make build-dev)
# just up-locbin
# BKMS_LOCAL_BINARY=/path/to/bkms-server just up-locbin

# 可选：启用真实部署类 API 测试
# 如果启用 Helm / Trpc / TAF 部署测试，需额外提供目标集群可拉取的 workload 镜像仓库地址（不带 tag），
# 且该 registry 中必须预先存在 1.0.0 / 2.0.0 两个 tag 的镜像，且两个 tag 的镜像都包含 sleep 命令。
export BKMS_APITEST_WORKLOAD_IMAGE=example.com/bkms/apitest-workload
RUN_HELM_DEPLOY_API_TESTS=1 RUN_TRPC_DEPLOY_API_TESTS=1 RUN_TAF_DEPLOY_API_TESTS=1 just up

# 实时查看 bruno 测试结果
just e2e-logs
just e2e-logs-tail 200    # 仅显示最近 200 行
just e2e-logs-failures    # 仅筛选失败用例、失败断言和直接原因

# 重启 bkms-server（修改镜像后想重新加载）
# just restart-bkms

# 清理：compose down -v --remove-orphans
just down
```

`just up` 会检查 kubeconfig 存在后调用 `docker-compose up -d`。容器到 K8s apiserver 的可达性由调用方（kind 启动方式 / 集群所在网络）保证。

`just up-locbin` 会额外叠加 `compose-replace-binary.yaml`，将 `BKMS_LOCAL_BINARY` 指向的宿主机二进制只读挂载到容器内的 `/usr/bin/bkms-server`。

这个 patch 会同时作用于 `bkms-server`、`bkms-worker`、`db-migrate` 和 `migration`，确保 webserver、worker、schema 迁移与内置组件加载都运行同一份本地代码。

该模式仍会使用 `BKMS_SERVER_IMAGE` 提供基础运行环境与静态资源；如果改动涉及镜像内的依赖、配置文件、assets 或其他非二进制内容，仍需重新构建镜像。

#### 使用 podman 替换 docker

如果本地使用 Podman，尤其是 rootless Podman，可以改用 Podman 专用入口：

```bash
cd tests/

# 启动 Podman E2E 栈
just up-podman

# 可选：使用本地二进制启动
# (cd .. && make build-dev)
# just up-podman-locbin
# BKMS_LOCAL_BINARY=/path/to/bkms-server just up-podman-locbin

# 清理 Podman E2E 栈
just down-podman
```

`just up-podman` / `just up-podman-locbin` 会使用 `podman compose`，并将 `kubernetes.default` 映射为 `host.containers.internal` 以触达宿主机上的 K8s apiserver。

#### CI 流水线脚本（参考）

CI 上的 kind 已配置为监听 `0.0.0.0:42099`，docker bridge 直接可达，无需任何额外的端口转发。三个步骤参见下文"提交后的 CI 脚本"。

### 常用操作

#### 新增测试用例

在开发了新的 API 后，我们需要添加新的测试用例，这可以通过两种方式来完成：bruno GUI 工具或新建 `.bru` 文件。相比之下，大部分情况下更推荐使用 GUI 工具，因为可方便完成 API 排序、整理等操作。

新测试用例可以直接以已有用例作为模板，一些注意点：

- 组织测试用例时，应该以用户场景（use-case）作为主体，比如“更新 normal 类型文件的 content 应该成功”是一个 case；
- 这意味着，一个测试有时需连续调用多个 API 来完成，你可以用子目录来组织这些 API。

#### 编写 JavaScript 脚本来扩展测试逻辑

bruno 在许多地方都支持了 JavaScript，可以编写代码来扩展测试逻辑，实现一些较为复杂的测试逻辑，比如：

- 在 collection 整体层面上，每次调用 API 前生成一个随机字母后缀，追加在需要新建的应用名尾部，以避免名称发生冲突；
- 在单个 API 层面上，请求结束后将 response 中的某字段存放为运行时变量中，供下个请求读取并使用；
- 使用 [Chai](https://www.chaijs.com/) 测试框架语法，在 Tests 阶段编写灵活的 BDD 断言语句
- ……

以上用法，你都可以在当前 collection 中找到示例。

### 常见的测试模式

整理一些常见的 API 测试模式。

#### 调用多次 API 的联合测试

通过在一个子目录中添加多个 API，并加以排序编排，来完整测试一个产品功能场景。比如在 `app_config_files/set_content_of_overlay_should_fail` 目录中，存放了三个 API 测试用例：

1. Create base normal file：创建一个 normal 类型的基础应用配置文件对象；
2. Create overlay file：创建一个 overlay 类型的应用配置文件，其 base 设置为步骤 1 所创建的普通文件；
3. Update content：尝试更新该 overlay file 的 content 字段（应当出错）。

不同测试用例通过 bruno 的 runtime vars 运行时变量来传递数据，比如，在第一个用例的 post-response 阶段，执行了以下代码：

```javascript
bru.setVar('currentBaseVFileID', res.body.item.id);
```

执行第二个 API 用例时，request.body 引用了该变量：

```json5
{
  "name": "vfile-nor-{{randomSuffix}}",
  "type": "overlay",
  "contentSourceType": "local",
  // 将第一次调用所创建的 ID 作为基础文件
  "baseAppConfigFileID": "{{currentBaseVFileID}}"
}
```

#### 预先确保某个应用已经存在

在某些场景中，API 需依赖一个已经存在的应用才能正常使用（比如创建应用的应用配置文件）。此时，可通过在 API 的 pre-request 阶段调用 common 模块中的工具函数来达成目的。

举个例子，在 bru 文件中添加以下代码：

```
script:pre-request {
  const common = require("./common")
  
  // Make sure the default helm application exists.
  await common.createHelmApp()
}
```

> 如果这个预配置是目录级别的，则放在目录的 folder.bru 中。

可以确保环境变量中定义的 `helmAppID/Name` 在 API 测试执行前，已被成功创建。更多详情可查看 common.js 文件源码。
