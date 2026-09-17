# bkms-cli app update Reference

`bkms-cli app update` 用于更新应用配置。目前支持 `build-config` 子命令，用于更新应用的构建配置（sourceType、代码仓库、镜像仓库或流水线等）。

## build-config

更新应用的构建配置。通过 YAML 文件描述新的构建配置，支持三种 sourceType：`codeRepository`（代码仓库）、`imageRegistry`（镜像仓库）、`pipeline`（蓝盾流水线）。

**重要**：`build-config` 的 YAML 字段名与 `app get` 返回的 `buildConfig` 字段名不同：

| `app get` 返回字段 | `build-config` 输入字段 |
|---|---|
| `repoBuildConfig` | `codeRepo` |
| `imageBuildConfig` | `image` |
| `pipelineBuildConfig` | `pipeline` |

### 常用场景

```bash
# 更新构建配置
bkms-cli app update build-config --app myapp -f build_config.yaml
```

### YAML 配置示例

#### 场景一：代码仓库 + 仓库 Dockerfile（适用于 trpc / taf 应用）

适合已有 Dockerfile 的仓库，平台直接使用仓库中的 Dockerfile 构建镜像。

```yaml
sourceType: codeRepository
tagConfig:
  type: custom
  customOpts:
    withRevision: false
    withBuildTime: true
codeRepo:
  # 代码库接入方式，如 TGit
  type: TGit
  # 蓝盾侧仓库别名
  repoAlias: myteam/myapp
  repoURL: https://git.example.com/myteam/myapp.git
  defaultBranch: master
  # Dockerfile 路径，空则使用仓库根目录
  dockerfile: ./Dockerfile
  dockerBuildArgs:
    # Docker 构建参数（可选）
    BUILD_ARG1: value1
  imageBuildMode: repositoryDockerfile
```

#### 场景二：代码仓库 + 平台通用构建（适用于 trpc 应用）

平台通过指定构建镜像和运行镜像自动构建，无需维护 Dockerfile。

```yaml
sourceType: codeRepository
tagConfig:
  type: custom
  customOpts:
    withRevision: false
    withBuildTime: true
codeRepo:
  type: TGit
  repoAlias: myteam/myapp
  repoURL: https://git.example.com/myteam/myapp.git
  defaultBranch: main
  imageBuildMode: platform
  platformBuildConfig:
    # 编译阶段基础镜像
    builderImage: example.com/build-images/golang:1.21-alpine
    # 运行阶段基础镜像
    runnerImage: example.com/base-images/alpine:3.18
    commands:
      # 编译前置命令（如安装依赖）
      preBuild: []
      # 编译命令（空则使用平台默认）
      build: []
      # 运行环境命令
      runtimeEnv: []
      # 启动命令（空则由 appModelSpec.command 控制）
      start: ""
```

#### 场景三：镜像仓库（适用于 helm / agones 应用）

直接使用已有镜像仓库，无需代码构建流程。

```yaml
sourceType: imageRegistry
tagConfig:
  type: custom
  customOpts:
    withRevision: false
    withBuildTime: true
image:
  # 镜像名（不含 tag）
  name: example.com/myteam/myapp
  # 仓库用户名（可选，公开仓库可省略），密码写入后加密存储，list 返回时脱敏
  username: myrepo-user
  # 仓库密码，写入后加密存储，list 返回时脱敏
  # password: xxxxxx
```

#### 场景四：蓝盾流水线（适用于 trpc 应用）

由蓝盾流水线负责构建，平台触发流水线并获取镜像。

```yaml
sourceType: pipeline
pipeline:
  # 蓝盾流水线 ID
  pipelineID: p-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  params:
    # 流水线参数（key-value）
    BKMS_IMAGE_NAME: myapp
    BKMS_IMAGE_REGISTRY: example.com/myteam
    BKMS_IMAGE_TAG: latest
    BKMS_REPO_ALIAS: myteam/myapp
    BK_CI_BUILD_MSG: build trigger message
```

### tagConfig 说明

| 字段 | 说明 |
|------|------|
| `type` | Tag 生成策略：`semver`（语义化版本，由平台递增）或 `custom`（自定义规则） |
| `customOpts.prefix` | 自定义 Tag 前缀（可为空） |
| `customOpts.withRevision` | 是否拼接代码版本（commit hash） |
| `customOpts.withBuildTime` | 是否拼接构建时间戳 |

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `-f, --file` | 是 | YAML spec 文件路径 |
