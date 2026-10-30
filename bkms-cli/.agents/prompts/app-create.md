# bkms-cli app create Reference

`bkms-cli app create` 用于从 YAML spec 文件创建应用。

支持的应用：

- 平台托管应用：用 `framework: trpc | taf | blank` + `language` 描述技术栈；
- Helm / Agones 应用：用 `type: helm | agones`。

应用 ID 由服务端生成（`<name>-<suffix>`），**不能指定**；YAML 里写 `id` 会直接报错。

## 常用场景

```bash
# 从 YAML spec 文件创建应用（使用默认工作空间）
bkms-cli app create -f app.yaml

# 显式指定工作空间
bkms-cli app create -f app.yaml --workspace ws-demo
```

创建成功后输出应用 ID、名称和类型：

```
✓ App created successfully
  ID:   my-app-a1b2
  Name: my-app
  Type: bkmsApp
```

## 完整 YAML 示例

### trpc + imageRegistry

```yaml
name: my-trpc-app
framework: trpc
language: go

buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-trpc-app:latest
    # 可选，私有仓库用户名；密码写入后加密存储
    username: myuser
    # 可选，私有仓库密码（加密存储）
    # password: secret

appModelSpec:
  command: ["/app/bin/server"]

# 框架配置文件：trpc / taf 必填，blank 不能传
frameworkConfigFile:
  name: trpc_go.yaml
  mountDir: /app/conf
  # content 可选，也可以在应用创建后通过配置文件接口维护
```

### trpc + codeRepository

```yaml
# tRPC 应用，使用代码仓库构建
name: my-trpc-app
framework: trpc
language: go

buildConfig:
  sourceType: codeRepository
  repoBuildConfig:
    type: TGit
    repoAlias: my-repo
    repoURL: https://github.com/my-team/my-trpc-app.git
    defaultBranch: main
    dockerfile: Dockerfile

appModelSpec:
  command: ["/app/bin/server"]
  args: ["--config", "/app/conf/trpc_go.yaml"]
  envVars:
    - key: ENV
      value: prod
      # description 为可选字段
      description: "Runtime environment"

frameworkConfigFile:
  name: trpc_go.yaml
  mountDir: /app/conf
  content: |
    server:
      service:
        - name: trpc.app.server.service
          ip: 0.0.0.0
          port: 8080
```

### trpc + pipeline（蓝盾流水线构建）

```yaml
# tRPC 应用，由蓝盾流水线负责构建
name: my-trpc-app
framework: trpc
language: go

buildConfig:
  sourceType: pipeline
  pipelineBuildConfig:
    pipelineID: p-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    params:
      BKMS_IMAGE_NAME: my-trpc-app
      BKMS_IMAGE_TAG: latest

frameworkConfigFile:
  name: trpc_go.yaml
  mountDir: /app/conf
```

### taf + imageRegistry

```yaml
# TAF 应用只支持 C++，language 固定为 cpp
name: my-taf-app
framework: taf
language: cpp

buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-taf-app:latest

appModelSpec:
  command: ["/usr/local/taf/bin/taf_server"]

frameworkConfigFile:
  name: config.conf
  mountDir: /usr/local/taf/conf
```

### blank（不使用框架）+ 代码仓库

```yaml
# blank 应用不依赖框架，language 可以是 go / cpp / nodejs / python
# 不能传 frameworkConfigFile；blank 也没有开发模式、平台通用构建、管理命令
name: my-python-app
framework: blank
language: python

buildConfig:
  sourceType: codeRepository
  repoBuildConfig:
    type: TGit
    repoAlias: my-repo
    repoURL: https://github.com/my-team/my-python-app.git
    defaultBranch: main
    dockerfile: Dockerfile

appModelSpec:
  command: ["python3", "/app/main.py"]
  envVars:
    - key: PORT
      value: "8000"
```

### helm + GitRepo

```yaml
# Helm 应用，使用 Git 仓库中的 Chart
name: my-helm-app
type: helm

buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-helm-app:v1.0.0

helmSpec:
  helmSource:
    repoType: GitRepo
    valueFiles:
      - values.yaml
      - values-prod.yaml
    gitRepoConfig:
      type: TGit
      repoAlias: my-chart-repo
      repoURL: https://github.com/my-team/helm-charts.git
      revision: main
      sourceDir: charts/my-app
```

### helm + HelmRepo

```yaml
# Helm 应用，使用 Helm 仓库中的 Chart
name: my-helm-app
type: helm

buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-helm-app:v1.0.0

helmSpec:
  helmSource:
    repoType: HelmRepo
    helmRepoConfig:
      repoURL: https://charts.example.com/stable
      chartName: my-chart
```

### helm + BCSRepo

```yaml
# Helm 应用，使用 BCS 仓库中的 Chart
name: my-helm-app
type: helm

buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-helm-app:v1.0.0

helmSpec:
  helmSource:
    repoType: BCSRepo
    bcsRepoConfig:
      projectCode: my-project
      repoName: my-bcs-repo
      chartName: my-chart
```

## 旧写法（兼容保留）

`type: trpc` / `type: taf` 搭配 `appModelSpec.trpcSpec` / `appModelSpec.tafSpec` 仍然可用，
CLI 会自动转换成 `framework` + `language` + `frameworkConfigFile` 后再提交，并打印一条弃用提示。
新脚本请使用上面的 `framework` 写法。

```yaml
# 旧写法，仍可用
name: my-trpc-app
type: trpc
buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: example.com/my-team/my-trpc-app:latest
appModelSpec:
  trpcSpec:
    language: go
    fileName: trpc_go.yaml
    filePath: /app/conf
```
